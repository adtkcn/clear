package config

import (
	"encoding/json"
	"fmt"
	"os"
)

var PORT = ":1090"

var WhiteDirList = []string{`C:\\ProgramData`, `C:\\Windows`, `C:\\Program Files`, `Microsoft`, `.vscode\extensions`, `HBuilderX`, `vendor`, `微信web开发者工具`, `\resources\app`, `支付宝小程序开发工具`, "$RECYCLE.BIN", "nodejs"}

var SearchDirList = []string{`\node_modules`, `Yarn\Cache`, `\AppData\Local\Microsoft\TypeScript`}

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
func ReadConfig() {
	var config Config
	ReadJsonFile("config.json", &config)
	WhiteDirList = removeDuplicateElement(append(WhiteDirList, config.WhiteDirList...))
	SearchDirList = removeDuplicateElement(append(SearchDirList, config.SearchDirList...))

	fmt.Println("排除目录", WhiteDirList)
	fmt.Println("查找目录", SearchDirList)
}
func init() {
	ReadConfig()
}
