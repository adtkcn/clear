package controller

import (
	"clear/config"
	"fmt"
	"log"
	"os"
	"strings"
)

func isWhite(dirName string) bool {
	flag := false
	for _, v := range config.WhiteDirList {
		//判断字符串dirName中是否包含个子串v。包含或者v为空则返回true

		if strings.Contains(strings.ToLower(dirName), strings.ToLower(v)) {
			flag = true
			return true
		}
	}
	return flag
}
func isDel(dirName string) bool {
	flag := false
	for _, v := range config.SearchDirList {
		if strings.Contains(strings.ToLower(dirName), strings.ToLower(v)) {
			flag = true
			return true
		}
	}
	return flag
}

// 递归扫描目录
func ScanDirs(dirName string) []string {
	files, err := os.ReadDir(dirName)
	var node_modules []string
	if err != nil {
		log.Println("读取目录错误", dirName)
		return node_modules
	}
	for _, file := range files {
		dir := dirName + string(os.PathSeparator) + file.Name()
		if isWhite(dir) { // 白名单不扫描
			continue
		}

		if isDel(dir) { // 目录需要删除
			BatchSendWs("ScanDirs", dir)
			node_modules = append(node_modules, dir)
			log.Println(dir)
			continue
		}
		if file.IsDir() {
			node_modules = append(node_modules, ScanDirs(dirName+string(os.PathSeparator)+file.Name())...)
		}
	}

	return node_modules
}

// 删除目录下所有文件和目录
func DeleteDir(dirName string) {
	fmt.Println("deleteDir", dirName)
	err := os.RemoveAll(dirName)
	if err != nil {
		log.Println(err)
		return
	}
	BatchSendWs("DeleteDir", dirName)

	// files, err := os.ReadDir(dirName)
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }
	// for _, file := range files {
	// 	dir := dirName + string(os.PathSeparator) + file.Name()
	// 	if file.IsDir() {
	// 		DeleteDir(dir)
	// 	} else {
	// 		os.Remove(dir)
	// 	}
	// }
	// log.Println("删除：", dirName)
	// os.Remove(dirName)
}
