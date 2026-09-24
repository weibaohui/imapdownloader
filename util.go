package main

import (
	"mime"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// version 程序版本，用于 IMAP ID 命令与日志。
// 用 var 而非 const，便于构建时通过 -ldflags "-X main.version=..." 注入版本号（如 git tag）
var version = "1.1.0"

func mkdirFor(file string) error {
	return os.MkdirAll(filepath.Dir(file), os.ModePerm)
}

func openEML(file string) (*os.File, error) {
	return os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0640)
}

// decodeHeader 解码 RFC2047 编码的邮件头(如主题)，失败返回原文与错误
func decodeHeader(s string) (string, error) {
	if !strings.Contains(s, "=?") {
		return s, nil
	}
	dec := new(mime.WordDecoder)
	return dec.DecodeHeader(s)
}

// mailParseDate 解析 RFC5322 邮件 Date 头
func mailParseDate(s string) (time.Time, error) {
	return mail.ParseDate(s)
}

// existingUIDs 扫描已下载 .eml 文件，提取文件名中的 UID 构建已下载集合。
// POP3 文件名形如 <subject>-<uid>.eml，UID 取 ".eml" 去除后最后一个 "-" 之后部分；
// 无 "-" 时整段 stem 视为 UID。
func existingUIDs(dir string, uidMap map[int]string) map[string]bool {
	set := make(map[string]bool)
	uidBySeq := make(map[string]bool, len(uidMap))
	for _, uid := range uidMap {
		if uid != "" {
			uidBySeq[uid] = true
		}
	}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".eml") {
			return nil
		}
		stem := name[:len(name)-4]
		uid := stem
		if idx := strings.LastIndex(stem, "-"); idx >= 0 {
			uid = stem[idx+1:]
		}
		if uid != "" {
			set[uid] = true
		}
		return nil
	})
	if len(uidBySeq) > 0 {
		// 仅保留与服务器 UIDL 匹配的项，避免误判
		filtered := make(map[string]bool, len(set))
		for uid := range set {
			if uidBySeq[uid] {
				filtered[uid] = true
			}
		}
		return filtered
	}
	return set
}

// suppressUnused 占位，避免 log 包未使用告警（按需保留）
var _ = log.Infof
