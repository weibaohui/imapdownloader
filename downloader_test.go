package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap"
)

// TestIMAPIDCommandWire 验证 IMAP ID 命令的线上字节格式符合 RFC2971
func TestIMAPIDCommandWire(t *testing.T) {
	cmd := &imap.Command{
		Name: "ID",
		Arguments: []interface{}{
			[]interface{}{
				"name", "imapdownloader",
				"version", version,
				"vendor", "weibaohui",
				"support-email", "kefu@188.com",
			},
		},
	}
	cmd.Tag = "A001"
	var buf bytes.Buffer
	w := imap.NewWriter(&buf)
	if err := cmd.WriteTo(w); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	got := strings.TrimRight(buf.String(), "\r\n")
	// 期望: A001 ID ("name" "imapdownloader" "version" "1.1.0" "vendor" "weibaohui" "support-email" "kefu@188.com")
	want := `A001 ID ("name" "imapdownloader" "version" "` + version + `" "vendor" "weibaohui" "support-email" "kefu@188.com")`
	if got != want {
		t.Errorf("ID 命令字节不匹配\n got: %s\nwant: %s", got, want)
	}
}

func TestSanitizeSubject(t *testing.T) {
	cases := map[string]string{
		"hello/world?":   "helloworld",
		"主题：测试（一）":       "主题测试一",
		`"quoted" <a@b>`: "quoteda@b",
		"【标签】文件|名*?<>\\": "标签文件名",
	}
	for in, want := range cases {
		if got := sanitizeSubject(in); got != want {
			t.Errorf("sanitizeSubject(%q)=%q want %q", in, got, want)
		}
	}
}

func TestMailStorePath(t *testing.T) {
	date := time.Date(2024, 3, 5, 9, 0, 0, 0, time.UTC)
	got := mailStorePath("/abs", "INBOX", date, "测试/主题?", "uid123")
	want := filepath.ToSlash(filepath.Join("/abs", "INBOX", "2024", "03", "测试主题-uid123.eml"))
	if got != want {
		t.Errorf("mailStorePath got=%s want=%s", got, want)
	}
}

func TestExistingUIDs(t *testing.T) {
	dir := t.TempDir()
	// 制造已存在文件，后缀为 uid
	mustWrite(t, filepath.Join(dir, "2024", "03"), "测试主题-uid123.eml", "x")
	mustWrite(t, filepath.Join(dir, "2024", "04"), "other-uid456.eml", "x")
	mustWrite(t, filepath.Join(dir, "noteml.txt"), "nope", "x")

	uidMap := map[int]string{1: "uid123", 2: "uid456", 3: "uid789"}
	set := existingUIDs(dir, uidMap)
	if !set["uid123"] {
		t.Error("应包含 uid123")
	}
	if !set["uid456"] {
		t.Error("应包含 uid456")
	}
	if set["uid789"] {
		t.Error("不应包含未下载的 uid789")
	}
}

func TestOptionsParseDate(t *testing.T) {
	o := &Options{Since: "2024-01-15", Before: "2024-02-01"}
	st, ok := o.SinceTime()
	if !ok {
		t.Fatal("Since 解析失败")
	}
	if st.Year() != 2024 || st.Month() != time.January || st.Day() != 15 {
		t.Errorf("Since 解析错误: %v", st)
	}
	bt, ok := o.BeforeTime()
	if !ok || bt.Day() != 1 {
		t.Errorf("Before 解析错误: %v", bt)
	}
}

func mustWrite(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
