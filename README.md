# imapdownloader

# 特点
* 可以将邮件从服务器端完整地下载到本地，包括邮件的正文内容以及所有附件，确保信息的完整性。
* 支持按照指定的文件夹前缀进行有针对性的邮件备份，方便用户根据自己的需求选择特定的邮箱文件夹进行备份操作。
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
### 3、查看备份结果
   下载的邮件会按日期存储在 **backup** 目录下，双击即可打开 `.eml` 文件。

### 4、备份文件再次备份
   如有需要，可以将`backup`文件夹压缩备份到您喜欢的云盘、U盘、硬盘上。


# 邮件备份工具对比
---
**市面上虽然有许多邮件备份工具，但如何选择开源、安全、功能完善的工具？** 本文将横向对比几款主流邮件备份工具，剖析它们的优劣势，并重点介绍`imapdownloader`的独特优势，帮助你做出明智的选择。

---

### **横向对比：主流 IMAP 邮件备份工具**

| 特性                  | **imapdownloader**        | **imapsync**              | **MailStore Home**         | **OfflineIMAP**            |
|-----------------------|---------------------------|---------------------------|----------------------------|----------------------------|
| **开源性**           | ✅ 完全开源               | ✅ 开源                   | ❌ 专有免费版              | ✅ 开源                    |
| **安全性**           | ✅ 只存储本地             | ✅ 仅传输加密数据         | ❌ 可能存储云端            | ✅ 仅本地操作              |
| **备份完整性**       | ✅ 包括邮件正文和附件     | ✅ 包括邮件正文和附件     | ✅ 支持附件与存档          | ✅ 支持完整邮件            |
| **文件夹定向备份**   | ✅ 支持文件夹前缀筛选     | ❌ 需要手动指定文件夹     | ✅ 部分文件夹筛选          | ✅ 复杂规则支持            |
| **多次运行处理**     | ✅ 已下载自动跳过         | ✅ 同步支持               | ❌ 每次重新下载            | ✅ 支持同步处理            |
| **断点续传**         | ✅ 支持断点重连           | ✅ 支持                  | ❌ 不支持                  | ✅ 支持复杂重连            |
| **存储格式**         | ✅ `.eml` 直接打开        | ❌ 单纯同步，不易读       | ❌ 专属格式                | ❌ 不易于普通用户直接阅读  |
| **易用性**           | ✅ 配置简单，轻量运行     | ❌ 命令较复杂，适合技术型 | ✅ 图形化界面，易于上手    | ❌ 配置复杂，需高级技能    |

---

### **为何选择`imapdownloader`？**

#### 1. **完全开源，安全可靠**
`imapdownloader` 是一款完全开源的工具，你可以在任何平台上轻松验证代码的安全性，确保没有隐私泄露的风险。此外，它仅将邮件下载到本地存储，而不涉及第三方云服务，最大限度地保障用户的数据安全。开源地址https://github.com/weibaohui/imapdownloader

#### 2. **完整备份，保留原始内容与附件**
不同于某些工具只备份邮件正文，`imapdownloader` 会将邮件的**正文**与**所有附件**一并下载，完整保留邮件信息。下载后，邮件以 **.eml** 格式存储，可以直接双击使用任意邮件客户端打开。

#### 3. **灵活备份，针对性强**
通过配置文件，用户可以按指定的**文件夹前缀**（如“存档”、“INBOX”或“已发送”）进行有针对性的备份，避免不必要的邮件下载，节省时间与存储空间。

#### 4. **多次运行，增量式备份**
`imapdownloader` 支持多次运行，同一个邮箱内已下载的邮件会自动跳过，避免重复备份。此外，若中途连接中断，程序可以继续执行，**支持断点续传**，无需重新开始备份。

#### 5. **简单易用，零学习成本**
只需一个 **config.yaml** 配置文件，便可轻松启动程序。工具轻量高效，用户无需复杂的命令或图形界面操作，让所有用户都能轻松上手。

#### 6. **便于本地检索，支持第三方工具索引**
备份的邮件按 **年月文件夹** 进行分类，文件名以 **主题+时间戳** 格式存储，便于后续检索与归档。此外，用户可以结合 **Everything** 等文件索引工具，实现秒速搜索备份邮件。

---

### **与竞品对比的核心优势**

1. **比 imapsync 更易用：** `imapdownloader` 提供简单直观的 YAML 配置文件，减少了手动指定文件夹的繁琐。
2. **比 MailStore 更安全：** MailStore 免费版为专有软件，可能涉及云端存储，而`imapdownloader`完全本地化，保障数据隐私。
3. **比 OfflineIMAP 更友好：** `imapdownloader` 将邮件存储为 **.eml** 格式，普通用户无需复杂客户端即可直接打开邮件。

---



## 反馈
* 有问题请提issue

