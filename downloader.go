package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	log.Println("已连接到服务器:", d.Options.Host)

	if err = d.Client.Login(d.Options.Username, d.Options.Password); err != nil {
		return
	}
	log.Println("已登录:", d.Options.Username)

	return
}

// 下载邮箱
func (d *Downloader) downloadAccountMailbox(ctx context.Context, mailbox string) (err error) {

	d.currentMailbox = mailbox
	status, err := d.Client.Select(mailbox, true)
	if err != nil {
		return
	}
	log.Printf("当前邮箱文件夹%s,总数%d \n", status.Name, status.Messages)

	if status.Messages == 0 {
		return
	}
	all := status.Messages
	dir := filepath.Join(d.Options.absDir, mailbox)
	log.Printf("%s邮箱文件夹下载存放位置: %s\n", mailbox, dir)
	count := int(all / 100)
	t1 := time.Now()
	for i := 0; i <= count; i++ {
		start := i*100 + 1
		end := (i + 1) * 100
		if int(all)-start < 100 {
			end = int(status.Messages)
		}
		log.Printf("\n\n正在分析第%d批:[%d~%d]\n\n", i+1, start, end)
		err = d.downloadMailsByRange(ctx, uint32(start), uint32(end))
		if err != nil {
			return
		}
	}
	t2 := time.Since(t1)
	log.Printf("下载耗时：%0.0f分钟", t2.Minutes())
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
			log.Println("枚举邮箱文件夹结束")
			return
		case box := <-chBoxes:
			if box == nil {
				continue
			}
			log.Println("发现邮箱文件夹: ", box.Name)
			for _, prefix := range d.Options.Prefixes {
				if strings.HasPrefix(box.Name, prefix) {
					log.Println("符合前缀条件:", box.Name)
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
		log.Printf("[%d~%d]分析下载队列出错:%s\n", start, end, err.Error())
		return
	}
	if seqDL.Empty() {
		log.Printf("[%d~%d]下载队列为空,跳过:\n", start, end)
		return
	}
	err = d.downloadMailList(ctx, seqDL)
	if err != nil {
		log.Printf("[%d~%d]下载队列出错:%s\n", start, end, err.Error())
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
				log.Printf("分析邮件: %s\n", msg.Envelope.Subject)
				existed, err := d.checkMailStorePathExisted(msg)
				if err != nil {
					log.Printf("检测路径出错: %s\n", err)
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
	log.Println("开始下载队列:", seqDL.String())
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
	log.Printf("存储邮件：%s\n", file)
	if err = os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return
	}
	r := msg.GetBody(&imap.BodySectionName{})
	if r == nil {
		log.Println("message does not have a body:", msg.Envelope.MessageId)
		return
	}
	var f *os.File
	if f, err = os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640); err != nil {
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("清理文件报错:%s\n", err.Error())
		}
	}(f)

	if _, err = io.Copy(f, r); err != nil {
		return
	}
	return
}

// 获取邮件存储路径
func (d *Downloader) getMailStorePath(msg *imap.Message) string {
	year := msg.Envelope.Date.Format("2006")
	month := msg.Envelope.Date.Format("01")
	subject := msg.Envelope.Subject
	subject = strings.Replace(subject, "“", "", -1)
	subject = strings.Replace(subject, "”", "", -1)
	subject = strings.Replace(subject, "\"", "", -1)
	subject = strings.Replace(subject, "（", "", -1)
	subject = strings.Replace(subject, "）", "", -1)
	subject = strings.Replace(subject, " ", "", -1)
	subject = strings.Replace(subject, "。", "", -1)
	subject = strings.Replace(subject, "，", "", -1)
	subject = strings.Replace(subject, "【", "", -1)
	subject = strings.Replace(subject, "】", "", -1)
	subject = strings.Replace(subject, "：", "", -1)
	subject = strings.Replace(subject, ":", "", -1)
	subject = strings.Replace(subject, "/", "", -1)
	subject = strings.Replace(subject, "、", "", -1)
	subject = strings.Replace(subject, "<", "", -1)
	subject = strings.Replace(subject, ">", "", -1)
	subject = strings.Replace(subject, "*", "", -1)
	subject = strings.Replace(subject, "\\", "", -1)
	subject = strings.Replace(subject, "?", "", -1)
	subject = strings.Replace(subject, "|", "", -1)
	dir := filepath.Join(d.Options.absDir, d.currentMailbox)
	tid := fmt.Sprintf("%d", msg.Envelope.Date.UnixMilli())
	return filepath.Join(dir, year, month, fmt.Sprintf("%s-%s.eml", subject, tid))
}

// 检查邮件存储路径是否存在
func (d *Downloader) checkMailStorePathExisted(msg *imap.Message) (existed bool, err error) {
	file := d.getMailStorePath(msg)
	exists, err := PathExists(file)
	if err != nil {
		return
	}
	if exists {
		log.Printf("× 跳过：%s", msg.Envelope.Subject)
		return exists, nil
	}
	log.Printf("√ 下载：%s", msg.Envelope.Subject)
	return
}
