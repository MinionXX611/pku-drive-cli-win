# Windows 原生 MVP 使用说明

`pku-drive.exe` 是此 fork 提供的 Windows 原生命令行工具，不依赖 WSL。当前 MVP 聚焦安全的
远端操作：登录、查看状态、浏览目录和删除网盘项目。

## 获取或构建

每次推送都会通过 GitHub Actions 构建 `pku-drive-windows-amd64` 制品。也可以在 Windows 上从
源码构建。需要 Go 1.22 或更高版本，以及 Git：

```powershell
git clone https://github.com/MinionXX611/pku-drive-cli-win.git
cd pku-drive-cli-win
go test -count=1 ./...
.\scripts\build-windows.ps1 -Version 0.3.0
.\bin\pku-drive.exe version
```

如果 Go 被安装到 `D:\Go` 但尚未加入 `PATH`，先在当前 PowerShell 会话执行：

```powershell
$env:Path = "D:\Go\bin;$env:Path"
```

构建结果是 `bin\pku-drive.exe`。可将这个文件复制到个人的工具目录，再通过 PowerShell 或
CMD 调用它；不要将配置目录或凭据文件与该程序一起复制。

## 请在终端中运行

这是控制台程序，没有图形界面。双击 `pku-drive.exe` 时，Windows 会创建临时控制台；程序因
没有接收到命令而输出 `command required` 并以退出码 2 结束，窗口随即关闭。这是预期行为，
不是崩溃。

在 PowerShell 中运行如下命令可以保留输出：

```powershell
cd D:\path\to\pku-drive-cli-win
.\bin\pku-drive.exe version
.\bin\pku-drive.exe status --json
```

未登录时，`status --json` 会返回 `login required`（退出码 3）；这表示还未保存登录凭据，并非
可执行文件无法启动。

若确实需要从快捷方式打开并保留窗口，可将快捷方式目标设为：

```text
C:\Windows\System32\cmd.exe /k "D:\path\to\pku-drive.exe version"
```

将路径替换为实际位置。日常使用 PowerShell 更便于复制包含中文、空格或特殊字符的网盘路径。

## 登录和基本命令

在 PowerShell 中，正常登录会打开默认浏览器，并在浏览器跳回本机 `127.0.0.1` 回调地址后自动
返回终端：

```powershell
.\bin\pku-drive.exe login
.\bin\pku-drive.exe status
.\bin\pku-drive.exe ls /
.\bin\pku-drive.exe ls '/个人文档/课题'
```

若浏览器无法完成本机回调，使用粘贴模式。完成统一身份认证后，将浏览器地址栏中的完整回调
URL 粘贴到命令正在等待的终端；该 URL 含一次性认证信息，不能分享或保存：

```powershell
.\bin\pku-drive.exe login --paste-callback
```

## 删除文件和目录

删除文件：

```powershell
.\bin\pku-drive.exe rm '/个人文档/课题/report.csv'
```

删除目录及其内容必须显式附加 `--recursive`：

```powershell
.\bin\pku-drive.exe rm '/个人文档/课题/旧数据' --recursive
```

工具会将服务端返回的删除结果打印到终端；通常网盘会将删除项目移入回收站，但这取决于服务端
策略。`rm` 不提供客户端二次确认，请先用 `ls` 核对绝对网盘路径，尤其不要对重要目录执行带
`--recursive` 的命令。网络中断后，先再次列目录确认远端结果，再决定是否重试。

## 支持范围和本地数据

Windows MVP 支持：

- `login`、`status`、`ls`、`rm`、`logout`、`version`

以下命令在 Windows 版本会直接提示“不支持”，不会退回到未经验证的本地文件读写逻辑：

- `mkdir`、`put`、`get`
- `ls --share`、`get --share`

配置、OAuth 凭据和锁文件位于 `%APPDATA%\pku-drive`；上传的临时状态目录预留在
`%LOCALAPPDATA%\pku-drive`。保存到磁盘的敏感配置内容使用 Windows DPAPI 绑定到当前 Windows
用户。不要上传、同步、提交或复制这些目录给其他用户。执行 `logout` 可撤销远端 token；若只
希望移除本机状态，可使用 `logout --local-only`。

机器、网络、认证或命令错误的详细处理，请参阅[故障排查](troubleshooting.md)。
