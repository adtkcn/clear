package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"changeme/config"
)

func isWhite(dirName string) bool {
	for _, v := range config.WhiteList() {
		//判断字符串dirName中是否包含子串v。包含或者v为空则返回true
		if strings.Contains(strings.ToLower(dirName), strings.ToLower(v)) {
			return true
		}
	}
	return false
}

func isDel(dirName string) bool {
	for _, v := range config.SearchList() {
		if strings.Contains(strings.ToLower(dirName), strings.ToLower(v)) {
			return true
		}
	}
	return false
}

// emitFunc 向前端推送事件（替代原websocket的BatchSendWs）
type emitFunc func(eventType string, path string)

// projectProofFiles 开发项目标志：出现其一即可确认整棵目录属于用户项目，
// 其 node_modules 是可以删除的构建产物
var projectProofFiles = []string{
	"package-lock.json", "npm-shrinkwrap.json", "yarn.lock",
	"pnpm-lock.yaml", "pnpm-workspace.yaml", ".git",
}

// appMarkerFiles 软件自带资源标志：Electron / NW.js / CEF 系桌面应用
// （VS Code、Cursor、Trae、Discord、Postman、RedisDesktopManager 等）的运行时文件
var appMarkerFiles = []string{
	"electron.exe", "electron", "nw.exe",
	"chrome_elf.dll", "libcef.dll", "libegl.dll",
	"icudtl.dat", "resources.pak", "v8_context_snapshot.bin", "snapshot_blob.bin",
}

// 从 node_modules 向上查找项目/软件标志的最大层数
const proofMaxLevels = 8

// dirEntries 缓存已探测过的目录条目，避免同一祖先目录被反复 ReadDir
var dirEntries sync.Map // map[string][]fs.DirEntry

func resetScanCache() {
	dirEntries.Range(func(k, _ any) bool {
		dirEntries.Delete(k)
		return true
	})
}

// listDir 带缓存的目录读取，读取失败按空目录处理（不作为任何判定依据）
func listDir(dir string) []fs.DirEntry {
	if v, ok := dirEntries.Load(dir); ok {
		return v.([]fs.DirEntry)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		files = nil
	}
	dirEntries.Store(dir, files)
	return files
}

func hasAnyName(names []string, name string) bool {
	for _, v := range names {
		if name == v {
			return true
		}
	}
	return false
}

// isAppResourceDir 判定目录是否为 Electron / NW.js / CEF 系应用的资源目录或安装根目录。
// 命中条件：含 *.asar / *.nw / app.asar.unpacked 等包体，或含 Chromium 运行时标志文件，
// 或同时含可执行体与 *.pak 资源包（VS Code、Trae 等应用的根目录形态）。
// 这类目录内的 node_modules 属于软件自身，删除会导致软件崩溃，整体跳过不扫描。
func isAppResourceDir(files []fs.DirEntry) bool {
	var hasExe, hasPak bool
	for _, f := range files {
		name := strings.ToLower(f.Name())
		if f.IsDir() {
			if name == "app.asar.unpacked" || strings.HasSuffix(name, ".nw") {
				return true
			}
			continue
		}
		if strings.HasSuffix(name, ".asar") || strings.HasSuffix(name, ".nw") {
			return true
		}
		if hasAnyName(appMarkerFiles, name) {
			return true
		}
		if strings.HasSuffix(name, ".exe") {
			hasExe = true
		}
		if strings.HasSuffix(name, ".pak") {
			hasPak = true
		}
	}
	return hasExe && hasPak
}

// hasProjectProof 目录内是否存在开发项目标志（锁文件或 .git）
func hasProjectProof(files []fs.DirEntry) bool {
	for _, f := range files {
		if hasAnyName(projectProofFiles, strings.ToLower(f.Name())) {
			return true
		}
	}
	return false
}

// isSoftwareDir 判定目录是否为"已安装软件"目录：自身不是 npm 项目（无 package.json），
// 但含有可执行体。同时识别 IDE 类布局 <软件根>\bin\*.exe（DevEco Studio、IntelliJ IDEA 等）。
func isSoftwareDir(dir string, files []fs.DirEntry) bool {
	for _, f := range files {
		if !f.IsDir() && strings.EqualFold(f.Name(), "package.json") {
			return false
		}
	}
	exeFound := func(entries []fs.DirEntry) bool {
		for _, f := range entries {
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".exe") {
				return true
			}
		}
		return false
	}
	if exeFound(files) {
		return true
	}
	// JetBrains / IDE 安装布局：软件根目录下 bin\ 内是可执行体
	return exeFound(listDir(filepath.Join(dir, "bin")))
}

// isProjectNodeModules 判定 parent 下的 node_modules 属于用户项目还是软件自带。
// 从 parent 逐层向上查找（不越过扫描根 stop，避免命中扫描范围外的系统目录）：
// 先遇到项目标志（锁文件/.git）=> 可删；
// 先遇到软件标志（Chromium 应用资源 / 可执行体）=> 软件自带，跳过；
// 两者都查不到时保留删除能力（应对锁文件被清理的遗留项目目录）。
func isProjectNodeModules(parent string, stop string) bool {
	cur := parent
	for i := 0; i < proofMaxLevels; i++ {
		files := listDir(cur)
		if hasProjectProof(files) {
			return true
		}
		if isAppResourceDir(files) || isSoftwareDir(cur, files) {
			return false
		}
		if strings.EqualFold(cur, stop) { // 已到达扫描根
			break
		}
		up := filepath.Dir(cur)
		if up == cur {
			break
		}
		cur = up
	}
	return true
}

// scanWorkers 并发读取目录的 worker 数。单个目录读取很慢时（停转的移动硬盘、
// 受保护的恢复分区）只占住一个 worker，不会让整棵扫描原地卡住
const scanWorkers = 8

// rootSkipNames 盘根第一层的系统/保留目录。这些目录的 ACL 通常只开放给
// TrustedInstaller/System，整棵走下来只会产生大量读失败与等待（F:\Windows 类
// 非系统盘上的旧安装尤其明显：WinSxS 数万条目、$PatchCache 逐个失败）
var rootSkipNames = map[string]bool{
	"recovery": true, "boot": true, "efi": true, "perflogs": true,
	"config.msi": true, "$winreagent": true, "$recycle.bin": true,
	"system volume information": true, "windowsapps": true, "$extend": true,
	// 盘根上的系统安装目录：白名单只写了 C:\Windows，其他盘的 Windows 会整棵被扫
	"windows": true, "windows.old": true, "$windows.~bt": true, "$windows.~ws": true,
	"documents and settings": true, "programdata": true,
}

// isDriveRoot 判断扫描起点是否为盘根（Windows 驱动器根 / UNC 共享根）或 Unix 根目录。
// 系统目录跳过只在盘根生效，避免误伤 D:\projects\boot 这类普通项目目录
func isDriveRoot(p string) bool {
	if p == "" {
		return false
	}
	if runtime.GOOS != "windows" {
		return filepath.Clean(p) == string(filepath.Separator)
	}
	cleaned := filepath.Clean(p)
	volume := filepath.VolumeName(cleaned) // "F:\\work" -> "F:"，"\\\\nas\\share\\x" -> "\\\\nas\\share"
	if volume == "" {
		return false
	}
	// 卷名之后不再有实际路径层即为盘根；Clean("F:") 会得到 "F:."，一并归入
	rest := strings.Trim(cleaned[len(volume):], string(filepath.Separator)+".")
	return rest == ""
}

// normalizeScanRoot 归一化用户提交的扫描路径。Windows 下 "F:" 经 Clean 会变成
// 驱动器相对路径 "F:."，拼出的子路径（"F:.Recovery"）含义飘移且干扰白名单匹配，
// 因此盘符补回尾分隔符，不依赖该盘当前工作目录
func normalizeScanRoot(p string) string {
	p = strings.TrimSpace(p)
	if runtime.GOOS == "windows" && len(p) == 2 && p[1] == ':' && p[0] != '\\' {
		return p + string(filepath.Separator)
	}
	return filepath.Clean(p)
}

// isTopLevelSystemDir 扫描起点为盘根时，第一层命中系统保留目录名则跳过。
// 名称全等匹配（不区分大小写），不用子串匹配，避免误伤同名项目
func isTopLevelSystemDir(isTop bool, name string) bool {
	return isTop && rootSkipNames[strings.ToLower(name)]
}

// maxScanDepth 递归深度上限。Windows 存在自引用联接（如 ProgramData\Application Data
// 指回 ProgramData），没有限制时会无限下降；正常项目目录很少超过 40 层
const maxScanDepth = 32

// scanTask 待处理目录及其深度
type scanTask struct {
	dir   string
	depth int
}

// scanner 并发遍历器：共享任务栈 + worker 轮转，避免“一个慢目录卡住全部”
type scanner struct {
	root  string
	emit  emitFunc
	isTop bool // 扫描起点是否为盘根，决定 rootSkipNames 是否生效

	mu      sync.Mutex
	cond    *sync.Cond
	queue   []scanTask
	pending int
	found   []string
}

func (s *scanner) push(dirs []scanTask) {
	s.mu.Lock()
	s.pending += len(dirs)
	s.queue = append(s.queue, dirs...)
	s.mu.Unlock()
	s.cond.Signal()
}

func (s *scanner) report(dir string) {
	s.mu.Lock()
	s.found = append(s.found, dir)
	s.mu.Unlock()
	s.send("ScanDirs", dir)
	log.Println(dir)
}

func (s *scanner) send(eventType string, data string) {
	if s.emit == nil { // 测试场景不需要推送事件
		return
	}
	s.emit(eventType, data)
}

// visit 处理单个目录，返回需要继续遍历的子目录（含深度）
func (s *scanner) visit(dir string, isTop bool, depth int) []scanTask {
	files, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			log.Println("无权限读取，跳过", dir) // 不转发前端，避免刷屏
		} else {
			log.Println("读取目录错误", dir, err)
			s.send("ScanError", dir)
		}
		return nil
	}

	if isAppResourceDir(files) { // 应用安装目录整体不扫描
		log.Println("跳过应用目录", dir)
		return nil
	}

	var children []scanTask
	for _, file := range files {
		// filepath.Join 规范化分隔符，避免根路径带尾分隔符时拼出 "C:\\Windows" 双斜杠，
		// 导致白名单匹配失效
		child := filepath.Join(dir, file.Name())
		if isWhite(child) { // 白名单不扫描
			continue
		}
		if !file.IsDir() {
			// 只处理真实目录：同名压缩包不误报；指向其他盘的链接/junction 形式的
			// node_modules 不列入——删除只会移除链接本身、不释放空间却弄坏项目，
			// 应到目标盘单独扫描
			continue
		}

		if isDel(child) { // 目录需要删除
			// node_modules 需要“开发项目”证据，避免删除软件自带依赖导致其崩溃
			if strings.EqualFold(file.Name(), "node_modules") && !isProjectNodeModules(dir, s.root) {
				log.Println("跳过软件自带目录", child)
				continue
			}
			s.report(child)
			continue
		}
		if isTopLevelSystemDir(isTop, file.Name()) {
			log.Println("跳过系统目录", child)
			continue
		}
		if depth+1 > maxScanDepth { // 自引用联接会导致无限下降，硬性兜底
			log.Println("超过最大深度，跳过", child)
			continue
		}
		children = append(children, scanTask{dir: child, depth: depth + 1})
	}
	return children
}

func (s *scanner) worker(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		s.mu.Lock()
		for len(s.queue) == 0 && s.pending > 0 {
			s.cond.Wait()
		}
		if len(s.queue) == 0 { // pending 已归零，没有更多任务
			s.mu.Unlock()
			return
		}
		dir := s.queue[len(s.queue)-1]
		s.queue = s.queue[:len(s.queue)-1]
		s.mu.Unlock()

		children := s.visit(dir.dir, s.isTop && dir.depth == 0, dir.depth)

		s.mu.Lock()
		s.pending += len(children) - 1
		s.queue = append(s.queue, children...)
		done := s.pending == 0
		s.mu.Unlock()

		if done {
			s.cond.Broadcast() // 唤醒全部 worker 退出
		} else {
			s.cond.Signal() // 有新任务，唤醒一个
		}
	}
}

// 递归扫描目录，root 为本次扫描根，作为软件标志向上探测的边界与盘根判定依据。
// 并发遍历，返回命中目录的顺序不保证
func scanDirs(root string, dirName string, emit emitFunc) []string {
	cleanRoot := normalizeScanRoot(root)
	s := &scanner{
		root:  cleanRoot,
		emit:  emit,
		isTop: isDriveRoot(cleanRoot),
	}
	s.cond = sync.NewCond(&s.mu)

	s.push([]scanTask{{dir: dirName, depth: 0}})

	var wg sync.WaitGroup
	for i := 0; i < scanWorkers; i++ {
		wg.Add(1)
		go s.worker(&wg)
	}
	wg.Wait()

	return s.found
}

// 删除目录下所有文件和目录
func deleteDir(dirName string, emit emitFunc) {
	fmt.Println("deleteDir", dirName)
	err := os.RemoveAll(dirName)
	if err != nil {
		log.Println(err)
		return
	}
	emit("DeleteDir", dirName)
}
