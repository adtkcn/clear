package main

import (
	"path/filepath"
	"strings"

	"changeme/config"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// wailsApp 由 main() 创建应用后注入，供服务向前端发事件
var wailsApp *application.App

// ScanEvent 扫描/删除进度事件（替代原websocket推送的消息结构）
type ScanEvent struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// ClearService 暴露给前端的目录清理服务
type ClearService struct {
	scanning bool
	scanRoot string // 本次扫描根，作为软件标志向上探测的边界
}

func (s *ClearService) emit(eventType string, path string) {
	if wailsApp == nil {
		return
	}
	wailsApp.Event.Emit("scanEvent", ScanEvent{Type: eventType, Data: path})
}

// StartScan 异步扫描指定目录，扫中的目录通过 scanEvent 逐个推送给前端
func (s *ClearService) StartScan(dir string) string {
	if s.scanning {
		return "扫描正在进行中"
	}
	if strings.TrimSpace(dir) == "" {
		return "请输入路径或者选择磁盘"
	}
	s.scanning = true
	// 扫描根既是 node_modules 归属判定的向上探测边界，也决定盘根系统目录跳过是否生效
	s.scanRoot = normalizeScanRoot(dir)
	config.ReadConfig()
	go func() {
		defer func() { s.scanning = false }()
		resetScanCache() // 清理上一次扫描的目录探测缓存
		scanDirs(s.scanRoot, s.scanRoot, s.emit)
		s.emit("ScanDone", dir)
	}()
	return "执行中"
}

// DeleteDirs 异步批量删除目录，每个目录删除完成后通过 scanEvent 推送
func (s *ClearService) DeleteDirs(dirs []string) string {
	if len(dirs) == 0 {
		return "没有需要删除的目录"
	}
	for _, dir := range dirs {
		go func(dir string) {
			// 双重保险：白名单目录拒绝删除
			if isWhite(dir) {
				s.emit("DeleteFailed", dir)
				return
			}
			// 双重保险：node_modules 必须能确认属于开发项目，避免误删软件自带依赖
			if strings.EqualFold(filepath.Base(dir), "node_modules") &&
				!isProjectNodeModules(filepath.Dir(dir), s.scanRoot) {
				s.emit("DeleteFailed", dir)
				return
			}
			deleteDir(dir, s.emit)
		}(dir)
	}
	return "执行中"
}
