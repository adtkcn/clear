package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var WhiteDirList = []string{
	`C:\ProgramData`,
	`C:\Windows`,
	`\Program Files`,            // 任意盘的常规软件安装目录（含 Electron 应用）
	`\Program Files (x86)`,      // 任意盘的 32 位软件安装目录
	`\AppData\Local\Programs`,   // 用户级安装的 Electron 应用（VS Code、Cursor 等）
	`\AppData\Roaming\npm`,      // npm 全局包目录，删除会破坏全局 CLI
	`\resources\`,               // Electron/NW.js 应用的资源目录（app.asar 所在层级）
	`\electron\`,                // 名为 electron 的中间目录（软件本体安装位置）
	`\nvm\`,                     // nvm 托管的 Node.js 版本目录
	`System Volume Information`, // 系统还原点，无读取权限
	`Config.Msi`,                // Windows Installer 回滚数据，无读取权限
	`WindowsApps`,               // 商店应用目录，ACL 受限
	`Microsoft`,
	`.vscode\extensions`,
	`HBuilderX`,
	`vendor`,
	`微信web开发者工具`,
	`支付宝小程序开发工具`,
	"$RECYCLE.BIN",
	"nodejs",
	`\DevEco Studio`,
	`\plugins\`,
	`\android\app\`,
}

var SearchDirList = []string{`\node_modules`, `Yarn\Cache`, `.pnpm-store`, `\AppData\Local\Microsoft\TypeScript`}

// mu 保护两个规则切片：扫描现在是多 worker 并发遍历，期间 ReadConfig 会重写切片，
// 直接读全局变量属于数据竞争。读取统一走下面的访问器
var mu sync.RWMutex

// WhiteList 返回排除目录列表快照（调用方只遍历不修改）
func WhiteList() []string {
	mu.RLock()
	defer mu.RUnlock()
	return WhiteDirList
}

// SearchList 返回待扫描目录列表快照
func SearchList() []string {
	mu.RLock()
	defer mu.RUnlock()
	return SearchDirList
}

type Config struct {
	WhiteDirList  []string `json:"white_dir_list"`
	SearchDirList []string `json:"search_dir_list"`
}

// 从json文件解析
func ReadJsonFile[T *Config](p string, v T) {

	data, err := os.ReadFile(p)
	if err != nil {
		fmt.Println(err)
		return
	}
	json.Unmarshal(data, &v)
}

// 数组去重
func removeDuplicateElement(arr []string) []string {
	result := make([]string, 0, len(arr))
	temp := map[string]struct{}{}
	for _, item := range arr {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// ReadConfig 从可执行文件同级目录读取 config.json，追加自定义的排除/扫描目录
func ReadConfig() {
	exePath, err := os.Executable()
	if err != nil {
		return
	}
	var config Config
	ReadJsonFile(filepath.Join(filepath.Dir(exePath), "config.json"), &config)
	mu.Lock()
	white := removeDuplicateElement(append(append([]string{}, WhiteDirList...), config.WhiteDirList...))
	search := removeDuplicateElement(append(append([]string{}, SearchDirList...), config.SearchDirList...))
	WhiteDirList, SearchDirList = white, search
	mu.Unlock()

	fmt.Println("排除目录", white)
	fmt.Println("查找目录", search)
}
