package main

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type Options struct {
	Dir      string   `yaml:"dir"`
	Host     string   `yaml:"host"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Prefixes []string `yaml:"prefixes"`
	absDir   string   // 绝对路径
}

func (o *Options) print() {
	log.Infof("======配置信息 开始======\n")
	log.Infof("用户名：%s\n", o.Username)
	log.Infof("服务器地址：%s\n", o.Host)
	for _, prefix := range o.Prefixes {
		log.Infof("邮箱文件夹：%s\n", prefix)
	}
	log.Infof("存储路径：%s\n", o.absDir)
	log.Infof("======配置信息 结束======\n")

}

func (o *Options) setAbsDir() {
	// 改为绝对路径
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
