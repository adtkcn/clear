package main

import (
	"os"
	"runtime"
	"syscall"
)

// GetDisks 返回当前可用的磁盘列表（Windows），其他平台返回根目录
func (s *ClearService) GetDisks() []string {
	if runtime.GOOS != "windows" {
		return []string{"/"}
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetLogicalDrives")
	ret, _, _ := proc.Call()
	bits := uint32(ret)

	var disks []string
	for i := 0; i < 26; i++ {
		if bits&(1<<i) == 0 {
			continue
		}
		letter := string(rune('A' + i))
		root := letter + `:\`
		// 光驱等无法读取的盘跳过
		if _, err := os.ReadDir(root); err != nil {
			continue
		}
		disks = append(disks, root)
	}
	return disks
}
