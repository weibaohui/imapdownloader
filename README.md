# imapdownloader

# 特点
* 可以将邮件从服务器端完整地下载到本地，包括邮件的正文内容以及所有附件，确保信息的完整性。
* 支持按照指定的文件夹前缀进行有针对性的邮件备份，方便用户根据自己的需求选择特定的邮箱文件夹进行备份操作。
* 最新版本为 v0.3，不断在功能和稳定性上进行优化和提升。
* 可多次运行，已下载的邮件会自动跳过。
* 下载过程中连接断了的话，可以再次执行。
* 本工具将遍历符合前缀的邮箱文件夹，逐个下载邮件，包括邮件内的附件。
  存储时，按照年月创建文件夹，使用邮件主题+时间戳的格式，保存为eml文件。
  查看时可以直接双击打开eml文件。
* 支持everything等搜索工具索引


原文链接：https://blog.csdn.net/zihuxinyu/article/details/144511612

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


## 反馈
* 有问题请提issue

