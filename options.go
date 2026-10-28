package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type Options struct {
	Dir      string   `yaml:"dir"`
	Host     string   `yaml:"host"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Prefixes []string `yaml:"prefixes"`

	// Protocol 取值 imap(默认) | pop3
	Protocol string `yaml:"protocol"`
	// Delete 仅 POP3 生效：下载成功后删除服务器上的对应邮件
	Delete bool `yaml:"delete"`

	// From 筛选发件人：子串匹配，不区分大小写，匹配发件人邮箱与姓名
	From string `yaml:"from"`
	// Since 筛选起始日期 YYYY-MM-DD（含），邮件 Date >= 当日 00:00
	Since string `yaml:"since"`
	// Before 筛选截止日期 YYYY-MM-DD（不含），邮件 Date < 当日 00:00
	Before string `yaml:"before"`

	absDir string // 绝对路径
}

func (o *Options) print() {
	log.Infof("======配置信息 开始======\n")
	log.Infof("协议：%s\n", o.EffectiveProtocol())
	log.Infof("用户名：%s\n", o.Username)
	log.Infof("服务器地址：%s\n", o.Host)
	for _, prefix := range o.Prefixes {
		log.Infof("邮箱文件夹：%s\n", prefix)
	}
	log.Infof("存储路径：%s\n", o.absDir)
	if o.From != "" || o.Since != "" || o.Before != "" {
		log.Infof("筛选-发件人：%s\n", o.From)
		log.Infof("筛选-起始日期(含)：%s\n", o.Since)
		log.Infof("筛选-截止日期(不含)：%s\n", o.Before)
	}
	if o.Protocol == "pop3" && o.Delete {
		log.Warnf("⚠️ 已开启 POP3 下载后删除服务器邮件，删除在 QUIT 后生效且不可恢复\n")
	}
	log.Infof("======配置信息 结束======\n")

}

// EffectiveProtocol 返回规范化后的协议名，缺省 imap
func (o *Options) EffectiveProtocol() string {
	p := strings.ToLower(strings.TrimSpace(o.Protocol))
	if p == "" {
		return "imap"
	}
	return p
}

// SinceTime 解析 Since 为当日 00:00 的本地时间
func (o *Options) SinceTime() (time.Time, bool) {
	return o.parseDate(o.Since)
}

// BeforeTime 解析 Before 为当日 00:00 的本地时间
func (o *Options) BeforeTime() (time.Time, bool) {
	return o.parseDate(o.Before)
}

// parseDate 按 YYYY-MM-DD 解析为当日本地 00:00
func (o *Options) parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		log.Warnf("日期解析失败[%s]: %s，忽略该筛选条件\n", s, err.Error())
		return time.Time{}, false
	}
	return t, true
}

func (o *Options) setAbsDir() {
	// 配置里已是绝对路径（如 /vol1/mail、C:\mail）则直接采用，
	// 不再拼到可执行文件目录下（filepath.Join 会把绝对路径当相对段拼坏）
	if filepath.IsAbs(o.Dir) {
		o.absDir = filepath.Clean(o.Dir)
		return
	}
	// 相对路径：相对可执行文件所在目录
	o.absDir = filepath.Join(GetCurrentDirectory(), o.Dir)
}

func GetCurrentDirectory() string {
	// 返回绝对路径  filepath.Dir(os.Args[0])去除最后一个元素的路径
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
	// 将\替换成/
	// return strings.Replace(dir, "\\", "/", -1)
}
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
