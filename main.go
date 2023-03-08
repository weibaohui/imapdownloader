package main

import (
	"context"
	"os"

	"github.com/emersion/go-imap/client"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func init() {
	// log.SetFormatter(&log.Formatter())
	log.SetOutput(os.Stdout)
	file, err := os.OpenFile("logrus.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}
	log.SetLevel(log.TraceLevel)
}
func main() {
	opts := &Options{}
	buf, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("读取配置文件出错:%s\n", err.Error())
	}
	err = yaml.Unmarshal(buf, opts)
	if err != nil {
		log.Fatalf("转换配置文件出错:%s\n", err.Error())
	}
	opts.setAbsDir()
	opts.print()
	ctx := context.Background()
	if err = DownloadByAccount(ctx, opts); err != nil {
		log.Printf("下载报错：%s\n", err.Error())
	}
}

// DownloadByAccount 按邮箱账户进行下载
func DownloadByAccount(ctx context.Context, opts *Options) (err error) {
	d, err := NewDownloader(opts)
	if err != nil {
		return
	}

	defer func(Client *client.Client) {
		err := Client.Logout()
		if err != nil {
			log.Printf("退出登录出错：%s\n", err.Error())
		}
	}(d.Client)

	// 获取邮箱文件夹，并按前缀进行匹配
	mailboxes, err := d.getPrefixMatchedMailBoxes(ctx)

	// 逐个文件夹下载
	for _, mailbox := range mailboxes {
		err := d.downloadAccountMailbox(ctx, mailbox)
		if err != nil {
			return err
		}
	}

	return
}
