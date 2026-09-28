package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"changeme/config"
)

// TestScanDirsSkipsSoftwareDirs 用日志中出现的真实软件形态验证过滤规则：
// Electron 应用、NW.js 应用、JetBrains 系 IDE、VS Code/Trae 系安装包、nvm 式 Node 运行时
func TestScanDirsSkipsSoftwareDirs(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("SearchDirList 使用反斜杠匹配，仅 Windows 有效")
	}
	root := t.TempDir()
	resetScanCache()

	// 1. Electron 应用资源目录：resources\app.asar + app.asar.unpacked\node_modules
	appRes := filepath.Join(root, "mqttx", "resources")
	mustMkdir(t, filepath.Join(appRes, "app.asar.unpacked", "node_modules", "pkg"))
	mustTouch(t, filepath.Join(appRes, "app.asar"))

	// 2. NW.js 应用：nw.exe + node_modules
	nwApp := filepath.Join(root, "legacyapp")
	mustMkdir(t, filepath.Join(nwApp, "node_modules", "pkg"))
	mustTouch(t, filepath.Join(nwApp, "nw.exe"))

	// 3. JetBrains 系 IDE（DevEco Studio / IntelliJ IDEA）：<根>\bin\xxx.exe + plugins\...\node_modules
	studio := filepath.Join(root, "devecostudio")
	mustMkdir(t, filepath.Join(studio, "bin"))
	mustTouch(t, filepath.Join(studio, "bin", "devecostudio64.exe"))
	plg := filepath.Join(studio, "plugins", "codelinter")
	mustMkdir(t, filepath.Join(plg, "node_modules", "pkg"))
	mustTouch(t, filepath.Join(plg, "package.json")) // 插件自带 package.json，不应被当成用户项目

	// 4. VS Code / Trae 系应用根目录：可执行体 + *.pak + 顶层 node_modules
	trae := filepath.Join(root, "traecn")
	mustMkdir(t, trae)
	mustMkdir(t, filepath.Join(trae, "node_modules", "pkg"))
	mustTouch(t, filepath.Join(trae, "trae.exe"))
	mustTouch(t, filepath.Join(trae, "resources.pak"))

	// 5. nvm 式 Node 运行时目录：node.exe + node_modules（删除会导致 Node 不可用）
	nodeVer := filepath.Join(root, "nodevers", "v18.20.5")
	mustMkdir(t, nodeVer)
	mustMkdir(t, filepath.Join(nodeVer, "node_modules", "pkg"))
	mustTouch(t, filepath.Join(nodeVer, "node.exe"))

	// 6. Adobe 宿主软件：根目录有 exe，依赖埋在 Required\...\node_modules
	ps := filepath.Join(root, "photoshop")
	mustMkdir(t, ps)
	mustTouch(t, filepath.Join(ps, "photoshop.exe"))
	gen := filepath.Join(ps, "required", "generator-builtin")
	mustMkdir(t, filepath.Join(gen, "node_modules", "pkg"))
	mustTouch(t, filepath.Join(gen, "package.json"))

	// 7. 真实项目：package.json + 锁文件，应被扫出
	proj := filepath.Join(root, "proj")
	mustMkdir(t, filepath.Join(proj, "node_modules", "left-pad"))
	mustTouch(t, filepath.Join(proj, "package.json"))
	mustTouch(t, filepath.Join(proj, "package-lock.json"))

	// 8. monorepo：锁文件在仓库根，子包 node_modules 也应被扫出
	mono := filepath.Join(root, "mono")
	mustMkdir(t, filepath.Join(mono, ".git"))
	mustMkdir(t, filepath.Join(mono, "packages", "sub", "node_modules", "a"))
	mustTouch(t, filepath.Join(mono, "packages", "sub", "package.json"))
	mustTouch(t, filepath.Join(mono, "pnpm-lock.yaml"))

	// 9. 锁文件已被清理的遗留项目：无项目标志也无软件标志，保留删除能力
	leftover := filepath.Join(root, "leftover")
	mustMkdir(t, filepath.Join(leftover, "node_modules", "pkg"))

	var found []string
	emit := func(eventType string, path string) {
		if eventType == "ScanDirs" {
			found = append(found, path)
		}
	}

	scanDirs(root, root, emit)

	want := map[string]bool{
		filepath.Join(proj, "node_modules"):                    true,
		filepath.Join(mono, "packages", "sub", "node_modules"): true,
		filepath.Join(leftover, "node_modules"):                true,
	}
	if len(found) != len(want) {
		t.Fatalf("期望扫出 %d 个项目目录，实际 %d 个: %v", len(want), len(found), found)
	}
	// 不依赖 ReadDir 的字母遍历顺序，按集合比对
	for _, f := range found {
		if !want[f] {
			t.Fatalf("不应被扫出的目录: %s", f)
		}
	}
}

// TestIsWhiteRejectsAppPaths 验证白名单覆盖软件安装路径与无权限的系统目录
func TestIsWhiteRejectsAppPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("白名单使用反斜杠路径，仅 Windows 有效")
	}
	config.ReadConfig()
	paths := []string{
		`C:\Program Files\SomeApp\node_modules`,
		`D:\Program Files (x86)\Tool\node_modules`,
		`C:\Users\xh\AppData\Local\Programs\cursor\resources\app\node_modules`,
		`C:\Users\xh\AppData\Roaming\npm\node_modules`,
		`D:\software\nvm\v22.15.0\node_modules`,
		`D:\Config.Msi`,
		`D:\System Volume Information`,
		`D:\WindowsApps\MutableBackup`,
	}
	for _, p := range paths {
		if !isWhite(p) {
			t.Errorf("应命中白名单: %s", p)
		}
	}
	for _, p := range []string{`E:\work\proj\node_modules`, `D:\work\my-electron-app\node_modules`} {
		if isWhite(p) {
			t.Errorf("项目 node_modules 不应命中白名单: %s", p)
		}
	}
}

func TestIsDriveRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("盘根判定依赖 Windows 卷名语法，非 Windows 环境下不测")
	}
	drive := filepath.VolumeName(t.TempDir()) // 如 "C:"
	for _, p := range []string{drive, drive + `\`, drive + `\.`, drive + `\..`} {
		if !isDriveRoot(p) {
			t.Errorf("应判定为盘根: %s", p)
		}
	}
	for _, p := range []string{filepath.Join(drive, "work", "proj"), drive + `\Windows`, `.`} {
		if isDriveRoot(p) {
			t.Errorf("不应判定为盘根: %s", p)
		}
	}
}

func TestNormalizeScanRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("盘符归一化仅在 Windows 下有差异")
	}
	// "F:" 直接 Clean 会得到驱动器相对路径 "F:.",必须补回分隔符
	if got := normalizeScanRoot(" F:\\ "); got != `F:\` {
		t.Errorf("盘符应归一化为 F:\\, got=%q", got)
	}
	if got := normalizeScanRoot(`E:\work\`); got != `E:\work` {
		t.Errorf("普通路径应去除尾分隔符, got=%q", got)
	}
	if !isDriveRoot(normalizeScanRoot("F:")) {
		t.Error("归一化后的盘符应被识别为盘根")
	}
}

func TestTopLevelSystemDirSkip(t *testing.T) {
	// 扫盘根时，第一层的恢复/系统分区直接跳过，不去 ReadDir
	for _, name := range []string{"Recovery", "recovery", "RECOVERY", "$WinREAgent", "Boot", "EFI"} {
		if !isTopLevelSystemDir(true, name) {
			t.Errorf("盘根系统目录应跳过: %s", name)
		}
	}
	// 非盘根（如 D:\projects）不启用该规则，否则名为 boot 的项目会被误跳
	for _, name := range []string{"Recovery", "Boot"} {
		if isTopLevelSystemDir(false, name) {
			t.Errorf("非盘根不应跳过: %s", name)
		}
	}
	// 全等匹配，子串命中不算
	for _, name := range []string{"project-recovery-ui", "bootscreen", "recovery-tool"} {
		if isTopLevelSystemDir(true, name) {
			t.Errorf("同名前缀的项目目录不应被跳过: %s", name)
		}
	}
}

// 并发遍历不得漏项：大量子目录扇出时 worker 池需全部跑完再结束
func TestScanDirsConcurrentFanout(t *testing.T) {
	root := t.TempDir()
	resetScanCache()
	const projects = 60
	for i := range projects {
		p := filepath.Join(root, fmt.Sprintf("proj%03d", i))
		mustMkdir(t, filepath.Join(p, "src"))
		mustTouch(t, filepath.Join(p, "package.json"))
		mustTouch(t, filepath.Join(p, "package-lock.json"))
		mustMkdir(t, filepath.Join(p, "node_modules", "left-pad"))
	}

	got := scanDirs(root, root, nil)
	if len(got) != projects {
		t.Fatalf("应并发扫出 %d 个 node_modules，实际 %d 个", projects, len(got))
	}
}

// 扫非盘根目录时，即使内部有 Recovery 目录也正常扫描，验证不会误伤
func TestScanDirsKeepsRecoveryWhenNotDriveRoot(t *testing.T) {
	root := t.TempDir()
	resetScanCache()
	p := filepath.Join(root, "Recovery")
	mustMkdir(t, filepath.Join(p, "node_modules", "pkg"))
	mustTouch(t, filepath.Join(p, "package.json"))
	mustTouch(t, filepath.Join(p, "package-lock.json"))

	got := scanDirs(root, root, nil)
	want := filepath.Join(p, "node_modules")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("非盘根时 Recovery 目录应正常扫描, got=%v", got)
	}
}

// 同名文件不得被列为清理目标（如 node_modules.7z 压缩包）
func TestScanDirsIgnoresNonDirectories(t *testing.T) {
	root := t.TempDir()
	resetScanCache()
	mustTouch(t, filepath.Join(root, "node_modules.7z"))
	mustTouch(t, filepath.Join(root, "node_modules.zip"))

	proj := filepath.Join(root, "real")
	mustMkdir(t, filepath.Join(proj, "node_modules", "pkg"))
	mustTouch(t, filepath.Join(proj, "package.json"))
	mustTouch(t, filepath.Join(proj, "package-lock.json"))

	got := scanDirs(root, root, nil)
	want := filepath.Join(proj, "node_modules")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("应只报告真实目录 %v, got=%v", want, got)
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustTouch(t *testing.T, p string) {
	t.Helper()
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}
