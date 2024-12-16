# imapdownloader
* 将邮件批量备份到本地文件夹。
* 最新版本 v0.3
# 使用说明
## 二进制包
* 链接: [https://pan.baidu.com/s/1U0t76mh6bi7Z9hC3t5aOxA](https://pan.baidu.com/s/1U0t76mh6bi7Z9hC3t5aOxA)
* 提取码: bun7 

## 1、填写配置文件 config.yaml
```yaml
#要导出的目录
dir: backup
#主机地址，必须是TLS加密的IMAP协议地址
host: imap.qq.com:993
#用户名，登录IMAP服务器的用户名
username: qq@qq.com
#密码，登录IMAP服务器的密码
password: password
prefixes:
 - 存档
 - 收件箱
 - INBOX
 - 已发送
```
### 2、运行
程序默认读取同目录下的config.yaml文件。
程序可多次运行，自动跳过已下载邮件
```shell
./imapdownloader
```

### 说明
* 本工具将遍历符合前缀的邮箱文件夹，逐个下载邮件，包括邮件内的附件。
存储时，按照年月创建文件夹，使用邮件主题+时间戳的格式，保存为eml文件。
查看时可以直接双击打开eml文件。
* 支持everything等搜索工具索引

## 反馈
* 有问题请提issue

