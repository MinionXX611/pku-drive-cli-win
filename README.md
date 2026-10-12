# 北大网盘轻量 CLI

`pku-drive` 是北大网盘命令行工具。Linux 版支持浏览器登录、目录浏览、文件上传与下载、
创建目录、删除，以及直接下载他人分享的文件。Windows 原生 MVP 支持登录、查看状态、
浏览目录与删除；系统中的每位用户使用自己的北大账号和凭据。Windows 版本是控制台程序，
请从 PowerShell 或 CMD 运行，而不是双击打开 `.exe`。

## 安装

### Debian / Ubuntu

```bash
curl -LO https://github.com/Findddx/pku-drive-cli/releases/download/v0.3.0/pku-drive-cli_0.3.0-1_amd64.deb
curl -LO https://github.com/Findddx/pku-drive-cli/releases/download/v0.3.0/SHA256SUMS
grep '  pku-drive-cli_0.3.0-1_amd64.deb$' SHA256SUMS | sha256sum -c -
sudo apt install ./pku-drive-cli_0.3.0-1_amd64.deb
pku-drive version
```

### 其他 Linux amd64 发行版

```bash
curl -LO https://github.com/Findddx/pku-drive-cli/releases/download/v0.3.0/pku-drive-cli-0.3.0-linux-amd64.tar.gz
curl -LO https://github.com/Findddx/pku-drive-cli/releases/download/v0.3.0/SHA256SUMS
grep '  pku-drive-cli-0.3.0-linux-amd64.tar.gz$' SHA256SUMS | sha256sum -c -
sudo tar -xzf pku-drive-cli-0.3.0-linux-amd64.tar.gz --strip-components=1 -C /
/usr/local/bin/pku-drive version
```

同一台机器选择一种系统安装方式即可。安装后所有用户都能运行 `pku-drive`，每位用户以
自己的普通 Linux 账号登录；日常命令无需 `sudo`。

### Windows（原生 MVP）

Windows 版本不需要 WSL。安装 Go 1.22 或更高版本后，在 PowerShell 中构建：

```powershell
git clone https://github.com/MinionXX611/pku-drive-cli-win.git
cd pku-drive-cli-win
$env:Path = "D:\Go\bin;$env:Path" # Go 安装在其他目录时相应调整
go test -count=1 ./...
.\scripts\build-windows.ps1 -Version 0.3.0
.\bin\pku-drive.exe version
```

如果 Go 安装在 `D:\Go`，上例中的 `Path` 设置可直接使用；安装在其他位置时，请改为对应的
`bin` 目录，或确保 `go` 已在 `PATH` 中。GitHub Actions 也会为每次推送构建
`pku-drive-windows-amd64` 制品。

`pku-drive.exe` 没有图形界面。双击它会以没有命令的方式启动，显示 `command required` 后
临时控制台立即关闭，表现为“闪退”；这不是崩溃。请在已打开的 PowerShell 或 CMD 中运行，
例如：

```powershell
cd D:\path\to\pku-drive-cli-win
.\bin\pku-drive.exe version
.\bin\pku-drive.exe login
```

MVP 只支持 `login`、`status`、`ls`、`rm`、`logout` 与 `version`。`mkdir`、`put`、`get`
和分享链接命令会明确提示当前 Windows 构建尚不支持，避免使用未经 Windows 文件系统安全
验证的上传/下载路径。

配置和令牌保存在 `%APPDATA%\pku-drive`。文件内容由 Windows DPAPI 绑定到当前用户加密；
不要复制该目录给其他 Windows 用户，也不要将其中内容提交到版本控制。

完整的 Windows 使用、登录与常见问题见 [Windows 使用说明](docs/windows.md)。

### 从源码构建

```bash
git clone https://github.com/Findddx/pku-drive-cli.git
cd pku-drive-cli
make test
make build
./bin/pku-drive version
```

## 登录

Windows PowerShell 中，`pku-drive.exe login` 会打开默认浏览器，并在本地
`127.0.0.1` 回调完成后返回终端。若浏览器不能访问该回调地址，可使用
`pku-drive.exe login --paste-callback`，将地址栏完整 URL 粘贴回终端。

服务器有桌面、VNC 或 X11 浏览器时：

```bash
pku-drive login
pku-drive status
```

通过 SSH 登录服务器、浏览器运行在本地电脑时：

```bash
pku-drive login --paste-callback
```

在本地浏览器打开终端显示的授权链接并完成统一身份认证。浏览器最后跳转到
`http://127.0.0.1:PORT/callback?...` 后，复制地址栏中的**完整 URL**，粘贴回 SSH 终端并
回车。页面显示无法连接不影响粘贴认证。回调 URL 含一次性认证信息，请勿分享、记录或写入
命令历史。

## 路径规则

- 网盘路径是以 `/` 开头的绝对路径，例如 `'/个人文档/课题/data.csv'`。
- 本地路径与普通 shell 命令相同，可以是相对路径或绝对路径。
- 路径含空格、中文或 shell 特殊字符时，用单引号包围。
- 分享链接内的路径相对于分享根目录，不以 `/` 开头，例如 `'数据/样本.csv'`。

## 命令速查

Windows MVP 请仅使用表中标为“Windows MVP”的命令。

| 操作 | 命令 |
|---|---|
| 登录（Windows MVP） | `pku-drive login [--paste-callback]` |
| 查看登录状态（Windows MVP） | `pku-drive status` |
| 列出网盘目录（Windows MVP） | `pku-drive ls [网盘目录]` |
| 创建目录 | `pku-drive mkdir '网盘目录' [--parents]` |
| 上传文件 | `pku-drive put 本地文件 '网盘文件或目录'` |
| 下载文件 | `pku-drive get '网盘文件' 本地文件或目录` |
| 浏览分享链接 | `pku-drive ls --share 分享链接 [分享内目录]` |
| 下载分享文件 | `pku-drive get --share 分享链接 本地目录 [分享内文件...]` |
| 删除（Windows MVP） | `pku-drive rm '网盘文件或目录' [--recursive]` |
| 注销（Windows MVP） | `pku-drive logout` |
| 查看版本（Windows MVP） | `pku-drive version` |

## 常用操作

### 查看目录

```bash
pku-drive ls /
pku-drive ls '/个人文档/课题'
```

### 上传文件

参数顺序是 `put 本地文件 '网盘文件或目录'`：

```bash
# 上传并指定网盘文件名
pku-drive put ./report.csv '/个人文档/课题/report.csv'

# 上传到已有网盘目录，保留文件名 report.csv
pku-drive put ./report.csv '/个人文档/课题/'

# 覆盖网盘中的同名文件
pku-drive put ./report.csv '/个人文档/课题/report.csv' --overwrite

# 大文件上传；中断后重复同一命令可继续已校验的分片
pku-drive put ./dataset.tar '/个人文档/课题/dataset.tar' --quiet
```

### 下载自己的网盘文件

参数顺序是 `get '网盘文件' 本地文件或目录`：

```bash
# 下载并指定本地文件名
pku-drive get '/个人文档/课题/report.csv' ./report.csv

# 下载到已有目录，保留文件名 report.csv
pku-drive get '/个人文档/课题/report.csv' ./downloads/

# 覆盖本地同名文件
pku-drive get '/个人文档/课题/report.csv' ./report.csv --overwrite
```

### 创建与删除目录

```bash
# 父目录已经存在
pku-drive mkdir '/个人文档/课题/数据'

# 同时创建缺少的父目录
pku-drive mkdir '/个人文档/课题/数据/2026' --parents

# 删除文件，通常移入网盘回收站
pku-drive rm '/个人文档/课题/report.csv'

# 删除目录及其中内容
pku-drive rm '/个人文档/课题/旧数据' --recursive
```

Windows MVP 的 `rm` 与 Linux 版使用相同的远端删除接口。命令不会显示本地二次确认：执行前请
逐字检查绝对网盘路径；删除目录必须显式传入 `--recursive`。正常情况下服务端会将项目移入
网盘回收站，但应以网盘服务的实际策略为准。不要将未确认的通配符、相对路径或重要目录用于
删除命令。

## 下载分享链接

公开分享（anonymous）可直接浏览和下载。组织范围分享（realname）先运行 `pku-drive login`
或 `pku-drive login --paste-callback`，工具会使用当前用户身份检查访问权限。

交互式选择：

```bash
mkdir -p ./downloads
pku-drive get --share 'https://disk.pku.edu.cn/link/...' ./downloads/
```

本地目标必须是已存在的目录。单文件分享会直接下载；文件夹分享每次显示当前目录的一层
内容，进入目录时再加载下一层，因此大型分享也能快速浏览。选择会在切换目录后保留，下载
时维持分享中的相对目录结构。批量下载按文件逐个校验和落盘；若中途失败，终端会报告已完成
数量，已完成文件保留在目标目录。重试时可只选择未完成文件，或确认覆盖后加入 `--overwrite`。

| 按键 | 操作 |
|---|---|
| `↑` / `↓` 或 `k` / `j` | 移动光标 |
| `→` 或 `l` | 进入目录 |
| `←`、`h` 或 `Backspace` | 返回上级目录 |
| `Space` | 选中或取消当前文件 |
| `Enter` | 确认并开始下载 |
| `q` 或 `Esc` | 取消 |

脚本和无交互终端先用 JSON 列目录，再按相对路径下载一个或多个文件：

```bash
pku-drive ls --share 'https://disk.pku.edu.cn/link/...' --json
pku-drive ls --share 'https://disk.pku.edu.cn/link/...' '数据' --json
pku-drive get --share 'https://disk.pku.edu.cn/link/...' ./downloads/ \
  'README.pdf' '数据/样本.csv' --json
```

当前版本处理无需提取码和手机验证的分享链接。相关错误处理见
[故障排查](docs/troubleshooting.md#分享链接)。

## JSON 输出

在命令末尾加入 `--json` 可获得结构化结果：

```bash
pku-drive status --json
pku-drive ls '/个人文档/课题' --json
pku-drive put ./report.csv '/个人文档/课题/report.csv' --json
```

脚本应同时检查进程退出码和 JSON 中的 `ok` 字段。诊断信息与登录授权链接输出到 stderr。

## 文档

- [故障排查与高级说明](docs/troubleshooting.md)
- [API 与兼容性说明](docs/api-notes.md)
- [安全策略](SECURITY.md)
- [参与贡献](CONTRIBUTING.md)

本项目是社区维护的非官方工具，与北京大学及爱数无隶属关系。
