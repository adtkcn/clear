# go-clear

由 [clear](../clear)（gin + websocket + 浏览器页面）迁移而来的 Wails v3 桌面应用，用于扫描并清理磁盘上的 `node_modules`、`Yarn\Cache`、`.pnpm-store` 等目录。

## 迁移对照

| 原 clear | go-clear (Wails v3) |
| --- | --- |
| gin HTTP `/scan`、`/deleteDir` 接口 | `ClearService.StartScan`、`ClearService.DeleteDirs` 方法绑定 |
| websocket `/ws` 推送扫描/删除进度 | `scanEvent` 应用事件（`ScanEvent{type, data}`） |
| 浏览器打开静态页面（element-plus） | 内嵌 Vue 3 + TypeScript 窗口 UI |
| 硬编码 C:\~K:\ 磁盘列表 | `ClearService.GetDisks` 动态枚举可用磁盘 |
| `util`（jwt/RSA 等未使用代码） | 未迁移（死代码） |

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

针对“扫到 `F:\Recovery` 一类的目录会卡住”的修复：

- **并发遍历**：扫描改为 8 个 worker 共享任务队列（`scanWorkers`）。以前是单 goroutine 深度优先，
  一个读得很慢或一个读不动的目录就会让整个结果列表原地停止；现在慢目录只占住一个 worker，
  其他目录的结果继续流入列表。
- **盘根系统目录直接跳过**：`Recovery`、`Windows`、`Windows.old`、`Boot`、`EFI`、`Config.Msi`、
  `System Volume Information`、`$Recycle.Bin`、`$WinREAgent`、`WindowsApps`、`ProgramData`、
  `Documents and Settings`、`PerfLogs`、`$Extend` 共 15 个目录名。
  仅当扫描起点是盘根（`F:\`）时按**目录名全等**生效，因此 `D:\projects\boot`、
  `...\my-recovery-page` 这类项目目录不受影响。以前白名单只写了 `C:\Windows`，
  导致非系统盘上的旧安装（WinSxS 数万条目、`Installer\$PatchCache$` 逐个失败）会被整棵遍历，
  这是卡顿的主因。
- **权限错误不刷屏**：`fs.ErrPermission` 归为“无权限读取，跳过”，只记日志、不再向前端发事件；
  其余 IO 错误仍发 `ScanError`。
- **递归深度上限 64**（`maxScanDepth`）：Windows 存在自引用联接（`ProgramData\Application Data`
  指回 `ProgramData`），没有限制会无限下降。
- **只处理真实目录**：`node_modules.7z`、`node_modules.zip` 等同名文件不再被列为可删；
  指向其他盘的链接/junction 形式的 `node_modules` 也不列入（删除只会移除链接本身、
  不释放空间却弄坏项目），需要到目标盘单独扫描。
- **规则切片读写加锁**：`config.WhiteList()` / `config.SearchList()` 受 `RWMutex` 保护，
  避免并发扫描与 `ReadConfig` 重写切片之间的数据竞争。

实测（扫 `F:\` 全盘，多个历史项目盘）：完成耗时 7~9s，命中 43 个可删目录，
`F:\Recovery`、`F:\Windows` 不再被读取，无错误日志洪水。

## 界面

- 无背景图：已去除模板的 `.bg` 图层与图片引用，底色为纯色 `--ink` 加一层 CSS 渐变光晕。
- 宽高自适应：面板整宽填满窗口，内边距、字号、行高统一用 `clamp()` 随窗口缩放；
  结果列表（`.card` / `.list`）占据窗口剩余高度并在内部滚动，`body` 固定为 `100dvh`
  所以列表不会把页面撑高出现窗口滚动条。实测 460×380 / 1000×618 / 1600×900
  三档均无横纵溢出，列表可视行数约 3 / 11 / 19 行。
- 窄窗口（≤560px）：工具栏换为两行（路径输入框独占一行，磁盘下拉与扫描按钮同行），
  并隐藏列表序号列。
- 窗口尺寸：初始 1000×618，新增 `MinWidth: 420` / `MinHeight: 360` 下限。
- `frontend/public/` 下的 `bg-desktop.jpg`、`bg-mobile.jpg`、`Inter-Medium.ttf`、`vue.svg`
  已无任何引用，但仍会被复制进 `dist` 并嵌入二进制（约 670KB），可直接删除瘦身。

## 安全说明

- 删除端会二次校验白名单与项目证据，不满足则拒绝删除；前端只能从列表中排除条目，无法强制删除。
- 若仍有软件漏排除，在 `config.json` 的 `white_dir_list` 中补充其目录名即可（不区分大小写、子串匹配）。
- 删除前请检查一下，有些软件也会存在 node_modules 目录，以避免软件崩溃。
