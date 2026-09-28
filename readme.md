# x-clear

一款 Windows 磁盘清理桌面工具，用于扫描并清理开发机上占空间的大头目录：`node_modules`、`Yarn\Cache`、`.pnpm-store` 等。

前端工具链装完依赖后动辄几百 MB、上千个碎文件，手动逐个盘找、逐个删非常费时。x-clear 会并发遍历指定磁盘或目录，把命中且确认属于开发项目的目录列成清单，勾选后批量删除；同时通过一系列规则避免误删已安装软件（IDE、Electron 应用等）自带的 `node_modules`。

## 功能特性

- **动态磁盘枚举**：自动列出当前可读的盘符（Windows 下通过 `GetLogicalDrives` 获取，光驱等不可读盘自动跳过），也可直接输入任意路径作为扫描起点。
- **并发扫描**：多 worker 共享任务队列遍历目录树，命中一个目录就实时推送到前端列表，慢目录不会阻塞整体进度。
- **软件目录保护**：识别应用安装包体、Chromium/CEF 运行时文件、IDE 布局等特征，跳过软件自带的 `node_modules`（详见下文过滤规则）。
- **卡顿治理**：跳过盘根系统目录（`Recovery`、`Windows`、`System Volume Information` 等），权限错误不刷屏，递归深度封顶，只处理真实目录而非链接/压缩包。
- **批量删除**：列表中勾选后批量删除，每个目录删除结果单独回报；删除端会再做一次白名单与项目证据校验。
- **可配置规则**：通过同级 `config.json` 自定义排除目录与待扫描目录名称。
- **自适应界面**：窗口宽高任意缩放，面板整宽填满、列表内部滚动，窄窗口下工具栏自动折行。

## 技术栈

| 层 | 技术 |
| --- | --- |
| 桌面应用框架 | [Wails v3](https://v3.wails.io)（beta），Go 后端 + 内嵌 WebView 桌面壳，前端产物通过 Go `embed` 打包进单一可执行文件 |
| 后端 | Go 1.25，`goroutine` + 共享任务队列实现并发遍历，`syscall` 调用 Win32 API 枚举磁盘 |
| 前端 | Vue 3 + TypeScript + Vite，UI 直接手写组件，不依赖组件库 |
| 前后端通信 | Wails 服务方法绑定（`ClearService`）+ 应用事件 `scanEvent`（`ScanEvent{type, data}`）推送扫描/删除进度 |
| 构建 | Wails Taskfile（`wails3 dev` / `wails3 build`），`vue-tsc` 做类型检查 |

## 使用方式

开发模式运行（热重载）：

```bash
wails3 dev
```

打包生产版本（产物在 `build/bin`）：

```bash
wails3 build
```

修改 Go 服务方法或注册事件后，需重新生成前端绑定：

```bash
wails3 generate bindings -ts
```

运行测试：

```bash
go test ./...
```

## 自定义排除和扫描目录

在可执行文件同级目录下新建 `config.json` 文件：

```json5
{
    "white_dir_list": [], // 不扫描的目录，不区分大小写
    "search_dir_list": [] // 待扫描的目录名称，不区分大小写
}
```

## 软件目录过滤规则

已安装的软件（IDE、Electron 应用等）内部也带有 `node_modules`，误删会导致软件崩溃。`scan.go` 采用两类通用规则过滤：

**1. 应用资源 / 安装根识别**（命中则整棵子树不扫描）

- 包体：`*.asar`、`*.nw`、`app.asar.unpacked`
- Chromium 运行时：`electron(.exe)`、`nw.exe`、`chrome_elf.dll`、`libcef.dll`、`libEGL.dll`、`icudtl.dat`、`resources.pak`、`v8_context_snapshot.bin`、`snapshot_blob.bin`
- 同一目录内同时存在 `*.exe` 与 `*.pak`（VS Code、Trae 等应用根目录的形态）

**2. node_modules 归属判定**

从 `node_modules` 的父目录向上逐层查找标志（不越过本次扫描根，最多 8 层）：

- 先遇到**项目标志** → 可删除：`package-lock.json`、`npm-shrinkwrap.json`、`yarn.lock`、`pnpm-lock.yaml`、`pnpm-workspace.yaml`、`.git`
- 先遇到**软件标志** → 跳过：目录内含 `*.exe`，或符合 IDE 布局的 `bin\*.exe`，且该层没有 `package.json`
- 两者都没有 → 仍列为可删除（应对锁文件已被清理的遗留项目）

该规则覆盖 JetBrains 系 IDE（DevEco Studio / IntelliJ IDEA 的 `plugins`、`sdk`、`tools`）、Adobe 宿主软件、nvm 托管的 Node 运行时，以及 Electron / NW.js / CEF 系桌面应用。

**3. 路径白名单**

`C:\ProgramData`、`C:\Windows`、`\Program Files`、`\Program Files (x86)`、`\AppData\Local\Programs`、`\AppData\Roaming\npm`、`\resources\`、`\electron\`、`\nvm\`，以及无读取权限、只产生噪音日志的 `System Volume Information`、`Config.Msi`、`WindowsApps`。

## 扫描性能与卡顿处理

- **并发遍历**：扫描由 8 个 worker（`scanWorkers`）共享任务队列。单 goroutine 深度优先时，一个读得很慢或读不动的目录会让结果列表原地停止；现在慢目录只占住一个 worker，其他目录的结果继续流入列表。
- **盘根系统目录直接跳过**：`Recovery`、`Windows`、`Windows.old`、`Boot`、`EFI`、`Config.Msi`、`System Volume Information`、`$Recycle.Bin`、`$WinREAgent`、`WindowsApps`、`ProgramData`、`Documents and Settings`、`PerfLogs`、`$Extend` 共 15 个目录名。仅当扫描起点是盘根（如 `F:\`）时按**目录名全等**生效，因此 `D:\projects\boot`、`...\my-recovery-page` 这类项目目录不受影响。非系统盘上的旧安装（WinSxS 数万条目、`Installer\$PatchCache$` 逐个失败）是遍历卡顿的主因。
- **权限错误不刷屏**：`fs.ErrPermission` 归为“无权限读取，跳过”，只记日志、不再向前端发事件；其余 IO 错误仍发 `ScanError`。
- **递归深度上限 64**（`maxScanDepth`）：Windows 存在自引用联接（`ProgramData\Application Data` 指回 `ProgramData`），没有限制会无限下降。
- **只处理真实目录**：`node_modules.7z`、`node_modules.zip` 等同名文件不再被列为可删；指向其他盘的链接/junction 形式的 `node_modules` 也不列入（删除只会移除链接本身、不释放空间却弄坏项目），需要到目标盘单独扫描。
- **规则切片读写加锁**：`config.WhiteList()` / `config.SearchList()` 受 `RWMutex` 保护，避免并发扫描与 `ReadConfig` 重写切片之间的数据竞争。

实测（扫 `F:\` 全盘，多个历史项目盘）：完成耗时 7~9s，命中 43 个可删目录，`F:\Recovery`、`F:\Windows` 不再被读取，无错误日志洪水。

## 界面说明

- 纯色底 + CSS 渐变光晕，无背景图，二进制体积更小。
- 内边距、字号、行高统一用 `clamp()` 随窗口缩放；结果列表占据窗口剩余高度并在内部滚动，`body` 固定 `100dvh`，列表再长也不会出现窗口滚动条。实测 460×380 / 1000×618 / 1600×900 三档均无横纵溢出。
- 窄窗口（≤560px）：工具栏换为两行（路径输入框独占一行，磁盘下拉与扫描按钮同行），并隐藏列表序号列。
- 窗口初始 1000×618（按黄金比例），下限 `MinWidth: 420` / `MinHeight: 360`。

## 安全说明

- 删除端会二次校验白名单与项目证据，不满足则拒绝删除；前端只能从列表中排除条目，无法强制删除。
- 若仍有软件漏排除，在 `config.json` 的 `white_dir_list` 中补充其目录名即可（不区分大小写、子串匹配）。
- 删除前请检查一下，有些软件也会存在 node_modules 目录，以避免软件崩溃。
