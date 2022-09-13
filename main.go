package main

import (
	"fmt"
	"os/exec"
	"runtime"

	"clear/config"
	"clear/controller"

	"github.com/gin-gonic/gin"
)

// //go:embed static/**
// var static embed.FS
type Dir struct {
	Dir []string `json:"dir"`
}

func main() {
	//F:\\OpenSourceProject\\x-clear
	// node_modules := controller.ScanDirs("F:\\")
	// // fmt.Println(node_modules)
	// for _, dir := range node_modules {
	// 	go func(dir string) {
	// 		// controller.DeleteDir(dir)
	// 	}(dir)
	// }
	// controller.DeleteDir("F:\\\\xiangheng\\node\\task\\node_modules\\axios")
	router := gin.Default()

	router.GET("/ws", controller.GinWebsocketHandler(controller.WsServer))

	router.GET("/init", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "成功!",
			"data": map[string][]string{
				"WhiteDirList":  controller.WhiteDirList,
				"SearchDirList": controller.SearchDirList,
			},
		})
	})
	router.GET("/scan", func(c *gin.Context) {
		dir := c.Query("dir")

		fmt.Println("scan dir", dir)
		go controller.ScanDirs(dir)

		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "执行中",
			"data": "",
		})
	})

	router.POST("/deleteDir", func(c *gin.Context) {
		var dirs Dir
		_ = c.ShouldBindJSON(&dirs)
		// dir := c.PostFormArray("dir")

		fmt.Printf("%#v", dirs)
		for _, dir := range dirs.Dir {
			go func(dir string) {
				controller.DeleteDir(dir)
			}(dir)
		}

		// go controller.DeleteDir(dir)

		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "执行中",
			"data": "",
		})
	})

	router.GET("/send", controller.SendWs)
	// api := router.Group("/api")
	// {
	// 	api.GET("/file/list", controller.List)
	// }

	router.Static("/static", "./static") //静态文件

	urlAddr := "http://127.0.0.1" + config.PORT + "/static"
	fmt.Println("请打开地址：", urlAddr)
	if runtime.GOOS == "windows" {
		exec.Command(`cmd`, `/c`, `start`, urlAddr).Start()
	} else if runtime.GOOS == "linux" {
		exec.Command(`xdg-open`, `https://www.jianshu.com`).Start()
	} else if runtime.GOOS == "darwin" {
		exec.Command(`open`, urlAddr).Start()
	}
	router.Run(config.PORT)

}
