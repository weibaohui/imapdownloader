package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
)

type Downloader struct {
	Client         *client.Client
	Options        *Options
	currentMailbox string
}

func NewDownloader(opts *Options) (d *Downloader, err error) {
	d = &Downloader{}
	d.Options = opts
	// 增强邮件编码探测能力
	imap.CharsetReader = charset.Reader
	cli, err := client.DialTLS(d.Options.Host, nil)
	if err != nil {
		return
	}
	d.Client = cli
	log.Info("已连接到服务器:", d.Options.Host)

	if err = d.Client.Login(d.Options.Username, d.Options.Password); err != nil {
		return
	}
	log.Info("已登录:", d.Options.Username)

	// 部分邮箱（网易 126/163 等）要求登录后先发送 IMAP ID 命令(RFC 2971)进行客户端标识，
	// 否则后续 SELECT/EXAMINE 会返回 "Unsafe Login" 而无法下载。此处尽力发送，失败仅告警不中断。
	d.sendIMAPID()

	return
}

// sendIMAPID 登录后发送 IMAP ID 命令进行客户端标识，解决网易 126/163 "Unsafe Login" 问题
func (d *Downloader) sendIMAPID() {
	cmd := &imap.Command{
		Name: "ID",
		Arguments: []interface{}{
			// 参数为括号包裹的 字段名/值 字符串对，go-imap 会自动加引号与括号
			[]interface{}{
				"name", "imapdownloader",
				"version", version,
				"vendor", "weibaohui",
				"support-email", "kefu@188.com",
			},
		},
	}
	status, err := d.Client.Execute(cmd, nil)
	if err != nil {
		log.Warnf("发送IMAP ID命令失败（可能服务器不支持）: %s", err.Error())
		return
	}
	if status != nil && status.Err() != nil {
		log.Warnf("IMAP ID命令返回错误: %s", status.Err().Error())
		return
	}
	log.Infof("已发送IMAP ID命令完成客户端标识")
}

// 下载邮箱
func (d *Downloader) downloadAccountMailbox(ctx context.Context, mailbox string) (err error) {

	d.currentMailbox = mailbox
	status, err := d.Client.Select(mailbox, true)
	if err != nil {
		return
	}
	log.Infof("当前邮箱文件夹%s,总数%d \n", status.Name, status.Messages)

	if status.Messages == 0 {
		return
	}
	all := status.Messages
	dir := filepath.Join(d.Options.absDir, mailbox)
	log.Infof("%s邮箱文件夹下载存放位置: %s\n", mailbox, dir)
	count := int(all / 100)
	t1 := time.Now()
	for i := 0; i <= count; i++ {
		start := i*100 + 1
		end := (i + 1) * 100
		if int(all)-start < 100 {
			end = int(status.Messages)
		}
		log.Infof("\n\n正在分析第%d批:[%d~%d]\n\n", i+1, start, end)
		err = d.downloadMailsByRange(ctx, uint32(start), uint32(end))
		if err != nil {
			return
		}
	}
	t2 := time.Since(t1)
	log.Infof("下载耗时：%0.0f分钟", t2.Minutes())
	return
}

// 获取匹配前缀的邮箱文件夹
func (d *Downloader) getPrefixMatchedMailBoxes(ctx context.Context) (mailboxes []string, err error) {
	chBoxes := make(chan *imap.MailboxInfo)
	done := make(chan error, 1)
	go func() {
		done <- d.Client.List("", "*", chBoxes)
	}()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err = <-done:
			log.Infof("枚举邮箱文件夹结束")
			return
		case box := <-chBoxes:
			if box == nil {
				continue
			}
			log.Infof("发现邮箱文件夹: %s", box.Name)
			for _, prefix := range d.Options.Prefixes {
				if strings.HasPrefix(box.Name, prefix) {
					log.Infof("符合前缀条件: %s", box.Name)
					mailboxes = append(mailboxes, box.Name)
					break
				}
			}
		}
	}
}

// 按批次分析并下载邮件
func (d *Downloader) downloadMailsByRange(ctx context.Context, start, end uint32) (err error) {
	seqDL, err := d.getDownloadMailList(ctx, start, end)
	if err != nil {
		log.Errorf("[%d~%d]分析下载队列出错:%s\n", start, end, err.Error())
		return
	}
	if seqDL.Empty() {
		log.Infof("[%d~%d]下载队列为空,跳过:\n", start, end)
		return
	}
	err = d.downloadMailList(ctx, seqDL)
	if err != nil {
		log.Errorf("[%d~%d]下载队列出错:%s\n", start, end, err.Error())
		return
	}
	return
}

// 获取下载列表
// 下载列表填充规则：若本地已存在文件，则跳过
func (d *Downloader) getDownloadMailList(ctx context.Context, start uint32, end uint32) (seqDL *imap.SeqSet, err error) {
	seq := new(imap.SeqSet)
	seq.AddRange(start, end)

	chMsg := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	go func() {
		done <- d.Client.Fetch(seq, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid}, chMsg)
	}()

	seqDL = new(imap.SeqSet)
	for {
		select {
		case <-ctx.Done():
			return seqDL, ctx.Err()
		case err = <-done:
			return
		case msg := <-chMsg:
			if msg != nil {
				log.Infof("分析邮件: %s\n", msg.Envelope.Subject)
				if !d.matchFilter(msg) {
					log.Infof("× 筛选过滤：%s\n", msg.Envelope.Subject)
					continue
				}
				existed, err := d.checkMailStorePathExisted(msg)
				if err != nil {
					log.Errorf("检测路径出错: %s\n", err)
					return seqDL, err
				}
				if !existed {
					seqDL.AddNum(msg.Uid)
				}

			}
		}
	}
}

// 下载邮件列表
func (d *Downloader) downloadMailList(ctx context.Context, seqDL *imap.SeqSet) (err error) {
	log.Infof("开始下载队列: %s", seqDL.String())
	chMsg := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	var section imap.BodySectionName

	go func() {
		done <- d.Client.UidFetch(seqDL, []imap.FetchItem{imap.FetchEnvelope, section.FetchItem()}, chMsg)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err = <-done:
			return
		case msg := <-chMsg:
			if msg != nil {
				if err = d.downloadMail(msg); err != nil {
					return
				}
			}
		}
	}
}

// 下载邮件
func (d *Downloader) downloadMail(msg *imap.Message) (err error) {
	file := d.getMailStorePath(msg)
	log.Infof("存储邮件：%s\n", file)
	if err = os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return
	}
	r := msg.GetBody(&imap.BodySectionName{})
	if r == nil {
		log.Errorf("message does not have a body: %s", msg.Envelope.MessageId)
		return
	}
	var f *os.File
	if f, err = os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640); err != nil {
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Errorf("清理文件报错:%s\n", err.Error())
		}
	}(f)

	if _, err = io.Copy(f, r); err != nil {
		return
	}
	return
}

// 获取邮件存储路径
func (d *Downloader) getMailStorePath(msg *imap.Message) string {
	return mailStorePath(d.Options.absDir, d.currentMailbox, msg.Envelope.Date, msg.Envelope.Subject, fmt.Sprintf("%d", msg.Envelope.Date.UnixMilli()))
}

// matchFilter 按配置的发件人/时间筛选条件判断邮件是否保留
func (d *Downloader) matchFilter(msg *imap.Message) bool {
	if d.Options == nil {
		return true
	}
	// 发件人筛选：子串不区分大小写，匹配邮箱地址与姓名
	if from := strings.TrimSpace(d.Options.From); from != "" {
		matched := false
		for _, a := range msg.Envelope.From {
			if a == nil {
				continue
			}
			email := strings.ToLower(fmt.Sprintf("%s@%s", a.MailboxName, a.HostName))
			if strings.Contains(email, strings.ToLower(from)) {
				matched = true
				break
			}
			if strings.Contains(strings.ToLower(a.PersonalName), strings.ToLower(from)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	// 时间筛选：Date 缺失则不满足时间条件（保守跳过）
	mailDate := msg.Envelope.Date
	if !mailDate.IsZero() {
		if t, ok := d.Options.SinceTime(); ok && mailDate.Before(t) {
			return false
		}
		if t, ok := d.Options.BeforeTime(); ok && !mailDate.Before(t) {
			return false
		}
	}
	return true
}

// checkMailStorePathExisted 检查邮件存储路径是否存在
func (d *Downloader) checkMailStorePathExisted(msg *imap.Message) (existed bool, err error) {
	file := d.getMailStorePath(msg)
	exists, err := PathExists(file)
	if err != nil {
		return
	}
	if exists {
		log.Infof("× 跳过：%s", msg.Envelope.Subject)
		return exists, nil
	}
	log.Infof("√ 下载：%s", msg.Envelope.Subject)
	return
}
