package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/charset"
	log "github.com/sirupsen/logrus"
)

// pop3Conn 手写的 POP3 over TLS(POP3S) 客户端，仅实现下载所需的最小命令集
type pop3Conn struct {
	conn net.Conn
	br   *bufio.Reader
}

// dialPOP3TLS 拨号 POP3S，读取服务器问候
func dialPOP3TLS(addr string) (*pop3Conn, error) {
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return nil, err
	}
	pc := &pop3Conn{conn: conn, br: bufio.NewReader(conn)}
	if _, err := pc.readLineOK(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("读取POP3问候失败: %w", err)
	}
	return pc, nil
}

// auth 发送 USER/PASS 登录
func (pc *pop3Conn) auth(user, pass string) error {
	if err := pc.simple("USER %s", user); err != nil {
		return err
	}
	return pc.simple("PASS %s", pass)
}

// stat 返回邮件数量
func (pc *pop3Conn) stat() (int, error) {
	line, err := pc.cmd("STAT")
	if err != nil {
		return 0, err
	}
	var count, size int
	if _, err := fmt.Sscanf(line, "%d %d", &count, &size); err != nil {
		return 0, fmt.Errorf("解析STAT失败[%s]: %w", line, err)
	}
	return count, nil
}

// uidl 返回序号→稳定UID 的映射
func (pc *pop3Conn) uidl() (map[int]string, error) {
	lines, err := pc.multiline("UIDL")
	if err != nil {
		return nil, err
	}
	m := make(map[int]string, len(lines))
	for _, l := range lines {
		var seq int
		var uid string
		if _, err := fmt.Sscanf(l, "%d %s", &seq, &uid); err == nil {
			m[seq] = uid
		}
	}
	return m, nil
}

// retr 取回指定序号的完整 RFC822 报文
func (pc *pop3Conn) retr(seq int) ([]byte, error) {
	lines, err := pc.multiline("RETR %d", seq)
	if err != nil {
		return nil, err
	}
	// 多行响应行已被去除 CRLF；按 RFC5322 报文需用 CRLF 还原
	var buf bytes.Buffer
	for i, l := range lines {
		if i > 0 {
			buf.WriteString("\r\n")
		}
		buf.WriteString(l)
	}
	return buf.Bytes(), nil
}

// dele 标记删除，QUIT 后生效
func (pc *pop3Conn) dele(seq int) error {
	return pc.simple("DELE %d", seq)
}

// quit 发送 QUIT，服务器提交删除并关闭连接
func (pc *pop3Conn) quit() error {
	err := pc.simple("QUIT")
	_ = pc.conn.Close()
	return err
}

// simple 发送单行命令并期望 +OK
func (pc *pop3Conn) simple(format string, args ...interface{}) error {
	_, err := pc.cmd(format, args...)
	return err
}

// cmd 发送命令，读取单行响应，返回 +OK 之后的正文
func (pc *pop3Conn) cmd(format string, args ...interface{}) (string, error) {
	if err := pc.writeLine(format, args...); err != nil {
		return "", err
	}
	return pc.readLineOK()
}

// multiline 发送命令，读取多行响应，自动处理点填充
func (pc *pop3Conn) multiline(format string, args ...interface{}) ([]string, error) {
	if err := pc.writeLine(format, args...); err != nil {
		return nil, err
	}
	if _, err := pc.readLineOK(); err != nil {
		return nil, err
	}
	var lines []string
	for {
		line, err := pc.br.ReadString('\n')
		if err != nil {
			if err == io.EOF && line == "" {
				break
			}
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		// 结束标记：单独的点
		if line == "." {
			break
		}
		// 点填充还原：以点开头的行去掉首字符
		if strings.HasPrefix(line, "..") {
			line = line[1:]
		}
		lines = append(lines, line)
	}
	return lines, nil
}

func (pc *pop3Conn) writeLine(format string, args ...interface{}) error {
	cmd := fmt.Sprintf(format, args...)
	_, err := fmt.Fprintf(pc.conn, "%s\r\n", cmd)
	return err
}

// readLineOK 读取一行响应，要求以 +OK 开头，否则返回错误
func (pc *pop3Conn) readLineOK() (string, error) {
	line, err := pc.br.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "+OK") {
		return "", fmt.Errorf("POP3错误: %s", line)
	}
	// 返回 +OK 之后的正文（去前缀与空格）
	if len(line) > 3 {
		return strings.TrimSpace(line[3:]), nil
	}
	return "", nil
}

// DownloadByPOP3Account 按 POP3 协议下载整个邮箱
func DownloadByPOP3Account(ctx context.Context, opts *Options) (err error) {
	pc, err := dialPOP3TLS(opts.Host)
	if err != nil {
		return
	}
	log.Infof("已连接到POP3服务器: %s", opts.Host)

	defer func() {
		_ = pc.quit()
	}()

	if err = pc.auth(opts.Username, opts.Password); err != nil {
		return
	}
	log.Infof("已登录: %s", opts.Username)

	count, err := pc.stat()
	if err != nil {
		return
	}
	log.Infof("POP3邮件总数: %d", count)
	if count == 0 {
		return
	}

	uidMap, err := pc.uidl()
	if err != nil {
		// 某些服务器不支持 UIDL，退化为按序号下载且不可跨会话去重
		log.Warnf("获取UIDL失败: %s，将按序号下载", err.Error())
	}

	mailbox := pop3MailboxName(opts)
	dir := filepath.Join(opts.absDir, mailbox)
	downloaded := existingUIDs(dir, uidMap)

	t1 := time.Now()
	downloadedNow := 0
	for seq := 1; seq <= count; seq++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		uid := uidMap[seq]
		if uid != "" {
			if _, ok := downloaded[uid]; ok {
				log.Infof("× 跳过(已下载): seq=%d uid=%s", seq, uid)
				continue
			}
		}
		log.Infof("下载邮件: seq=%d/%d", seq, count)
		raw, e := pc.retr(seq)
		if e != nil {
			log.Errorf("RETR %d 失败: %s", seq, e.Error())
			err = e
			return
		}
		if e = savePOP3Mail(opts, mailbox, uid, raw); e != nil {
			log.Errorf("保存邮件失败 seq=%d: %s", seq, e.Error())
			err = e
			return
		}
		downloadedNow++
		if uid != "" {
			downloaded[uid] = true
		}
		if opts.Delete {
			if e := pc.dele(seq); e != nil {
				log.Errorf("DELE %d 失败: %s", seq, e.Error())
				err = e
				return
			}
			log.Infof("已标记删除服务器邮件 seq=%d", seq)
		}
	}
	log.Infof("POP3下载完成，本次下载 %d 封，耗时 %0.0f秒", downloadedNow, time.Since(t1).Seconds())
	return
}

// pop3MailboxName POP3 只有一个邮箱，用配置首个前缀或 "INBOX"
func pop3MailboxName(opts *Options) string {
	if len(opts.Prefixes) > 0 && opts.Prefixes[0] != "" {
		return opts.Prefixes[0]
	}
	return "INBOX"
}

// savePOP3Mail 解析报文 Date/Subject 生成存储路径并写入 .eml
func savePOP3Mail(opts *Options, mailbox, uid string, raw []byte) error {
	// 增强邮件编码探测能力
	message.CharsetReader = charset.Reader
	date, subject := parseEMLDateSubject(raw)
	suffix := uid
	if suffix == "" {
		// 无 UIDL 时退化为时间戳+序号，尽量保证唯一
		suffix = fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	file := mailStorePath(opts.absDir, mailbox, date, subject, suffix)
	log.Infof("存储邮件：%s", file)
	if err := mkdirFor(file); err != nil {
		return err
	}
	f, err := openEML(file)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	// 还原 CRLF 报文后写入；raw 已是 CRLF 文本
	if _, err := f.Write(raw); err != nil {
		return err
	}
	return nil
}

// parseEMLDateSubject 从原始报文中解析 Date 与 Subject 头，失败回退当前时间/空主题
func parseEMLDateSubject(raw []byte) (time.Time, string) {
	entity, err := message.Read(bytes.NewReader(raw))
	if err != nil {
		return time.Now(), ""
	}
	h := entity.Header
	subject := h.Get("Subject")
	if dec, derr := decodeHeader(subject); derr == nil && dec != "" {
		subject = dec
	}
	dateStr := h.Get("Date")
	var t time.Time
	if dateStr != "" {
		if dt, derr := mailParseDate(dateStr); derr == nil {
			t = dt
		}
	}
	if t.IsZero() {
		t = time.Now()
	}
	return t, subject
}
