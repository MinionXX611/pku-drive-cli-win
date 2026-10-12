# pku-drive-cli-win

北大网盘 Windows 原生命令行 MVP。它面向 Windows 用户提供浏览器登录、查看账号状态、浏览网盘
目录、删除文件或目录，以及注销功能。

本项目基于 [Findddx/pku-drive-cli](https://github.com/Findddx/pku-drive-cli) 开发，专注于 Windows 原生支持。

这是社区维护的非官方工具，与北京大学及爱数无隶属关系。

## 支持范围

当前 Windows MVP 支持以下命令：

| 命令 | 用途 |
|---|---|
| `login` | 在默认浏览器中完成登录，或粘贴本机回调 URL |
| `status` | 查看当前登录状态 |
| `ls` | 列出网盘目录 |
| `rm` | 删除网盘文件；使用 `--recursive` 删除目录树 |
| `logout` | 撤销远端登录状态，或移除本机凭据 |
| `version` | 显示版本与构建信息 |

`mkdir`、`put`、`get` 和分享链接相关命令尚不在 Windows MVP 的支持范围内。它们会返回
`command is not supported in this build`，不会执行未经 Windows 文件系统安全验证的操作。

## 获取与构建

每次推送均由 GitHub Actions 构建 `pku-drive-windows-amd64` 制品。也可以从源码构建；需要
Git、PowerShell 和 Go 1.22 或更高版本：

```powershell
git clone https://github.com/MinionXX611/pku-drive-cli-win.git
cd pku-drive-cli-win
go test -count=1 ./...
.\scripts\build-windows.ps1 -Version 0.3.0
.\bin\pku-drive.exe version
```

若 Go 安装在 `D:\Go` 但未加入 `PATH`，先在当前 PowerShell 会话运行：

```powershell
$env:Path = "D:\Go\bin;$env:Path"
```

构建生成的文件为 `bin\pku-drive.exe`。可将它复制到个人工具目录后从 PowerShell 或 CMD 调用。

## 从终端运行

`pku-drive.exe` 是控制台程序，不是图形应用。请不要双击打开：双击会因没有传入命令而显示
`command required`，随后临时控制台关闭，表现为“闪退”。这是预期行为，不是程序崩溃。

请在已打开的 PowerShell 或 CMD 中运行，例如：

```powershell
cd D:\path\to\pku-drive-cli-win
.\bin\pku-drive.exe version
.\bin\pku-drive.exe status --json
```

在没有登录时，`status --json` 返回 `login required`（退出码为 3），表示需要登录，并非启动
失败。若要从快捷方式运行并保留窗口，可以将目标设为：

```text
C:\Windows\System32\cmd.exe /k "D:\path\to\pku-drive.exe version"
```

将其中路径替换为实际的可执行文件路径。

## 登录与浏览目录

正常登录会打开默认浏览器。完成统一身份认证后，浏览器会跳回本机 `127.0.0.1`，终端将自动
继续：

```powershell
.\bin\pku-drive.exe login
.\bin\pku-drive.exe status
.\bin\pku-drive.exe ls /
.\bin\pku-drive.exe ls '/个人文档/课题'
```

如果浏览器无法访问本机回调地址，请使用粘贴模式：

```powershell
.\bin\pku-drive.exe login --paste-callback
```

完成认证后，将浏览器地址栏中完整的回调 URL 粘贴到正在等待的终端。该 URL 包含一次性认证
信息，不能分享、保存或写入命令历史。

## 删除文件和目录

网盘路径使用以 `/` 开头的绝对路径。包含空格、中文或特殊字符的路径应使用单引号：

```powershell
# 删除文件
.\bin\pku-drive.exe rm '/个人文档/课题/report.csv'

# 删除目录及其内容；必须显式传入 --recursive
.\bin\pku-drive.exe rm '/个人文档/课题/旧数据' --recursive
```

`rm` 不提供客户端二次确认。执行前请先用 `ls` 核对完整路径，尤其应谨慎使用 `--recursive`。
通常网盘服务会将项目移入回收站，但实际结果以服务端策略为准。若网络在删除过程中中断，先
重新列目录确认远端结果，再决定是否重试。

## 本机凭据与安全

配置、OAuth 凭据和锁文件保存在 `%APPDATA%\pku-drive`；临时状态目录预留在
`%LOCALAPPDATA%\pku-drive`。保存到磁盘的敏感配置内容由 Windows DPAPI 绑定到当前 Windows
用户保护。

不要复制、同步、提交或发送这些目录中的文件给其他用户。使用 `logout` 注销并尝试撤销远端
token；只需清除本机凭据时，使用 `logout --local-only`。

## 检查与问题排查

提交变更前可运行：

```powershell
go test -count=1 ./...
go vet ./...
.\scripts\build-windows.ps1 -Version dev
.\bin\pku-drive.exe version
```

更多 Windows 操作细节和故障处理见 [Windows 使用说明](docs/windows.md) 与
[故障排查](docs/troubleshooting.md)。请勿在 Issue、日志或截图中提供认证回调 URL、token、
Cookie、凭据文件或个人网盘路径。
