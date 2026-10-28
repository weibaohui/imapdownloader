package main

import (
	"path/filepath"
	"testing"
)

// config.yaml 里 dir 写绝对路径时必须原样采用，不得拼到可执行文件目录下
func TestSetAbsDirAbsolute(t *testing.T) {
	abs := t.TempDir() // 平台原生的绝对路径
	o := &Options{Dir: abs}
	o.setAbsDir()
	if o.absDir != filepath.Clean(abs) {
		t.Errorf("absolute dir mangled: got=%s want=%s", o.absDir, filepath.Clean(abs))
	}
}

// 相对路径仍相对可执行文件所在目录解析
func TestSetAbsDirRelative(t *testing.T) {
	o := &Options{Dir: "mail"}
	o.setAbsDir()
	want := filepath.Join(GetCurrentDirectory(), "mail")
	if o.absDir != want {
		t.Errorf("relative dir: got=%s want=%s", o.absDir, want)
	}
}
