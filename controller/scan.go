package controller

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var WhiteDirList = []string{`C:\\ProgramData`, `C:\\Windows`, `C:\\Program Files`, `Microsoft`, `.vscode\extensions`, `HBuilderX`, `vendor`, `微信web开发者工具`, `\resources\app`, `支付宝小程序开发工具`}

var SearchDirList = []string{`\node_modules`, `Yarn\Cache`, `\AppData\Local\Microsoft\TypeScript`}

func isWhite(dirName string) bool {
	flag := false
	for _, v := range WhiteDirList {
		//判断字符串dirName中是否包含个子串v。包含或者v为空则返回true

		if strings.Contains(strings.ToLower(dirName), strings.ToLower(v)) {
			// fmt.Println(strings.ToLower(dirName), strings.ToLower(v))
			// fmt.Println(strings.Contains(strings.ToLower(dirName), strings.ToLower(v)))
			flag = true
			return true
		}
	}
	return flag
}
func isDel(dirName string) bool {
	flag := false
	for _, v := range SearchDirList {
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
