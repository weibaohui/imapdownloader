package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// sanitizeSubject 清洗邮件主题，去除文件系统不允许的字符，用于生成文件名
func sanitizeSubject(subject string) string {
	// 首尾空白（含折叠头残留的 \r\n、全角空格等 Unicode 空白）一律去掉：
	// Windows 下结尾空白/换行属非法文件名字符；此处中心化处理，IMAP/POP3 两个调用方同享
	subject = strings.TrimSpace(subject)
	repl := []string{
		"“", "", "”", "", "\"", "",
		"（", "", "）", "", " ", "",
		"。", "", "，", "",
		"【", "", "】", "",
		"：", "", ":", "",
		"/", "", "、", "",
		"<", "", ">", "",
		"*", "", "\\", "",
		"?", "", "|", "",
	}
	r := strings.NewReplacer(repl...)
	return r.Replace(subject)
}

// mailStorePath 生成邮件存储路径：<absDir>/<mailbox>/<年>/<月>/<subject>-<suffix>.eml
// IMAP 侧 suffix 用时间戳毫秒；POP3 侧 suffix 用稳定 UID，便于跨会话去重。
func mailStorePath(absDir, mailbox string, date time.Time, subject, suffix string) string {
	if suffix == "" {
		suffix = fmt.Sprintf("%d", time.Now().UnixMilli())
	}
	year := date.Format("2006")
	month := date.Format("01")
	dir := filepath.Join(absDir, mailbox)
	return filepath.Join(dir, year, month, fmt.Sprintf("%s-%s.eml", sanitizeSubject(subject), suffix))
}
