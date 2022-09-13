package main

import (
	"fmt"

	"clear/config"
	"clear/controller"
	"clear/util"

	"github.com/gin-gonic/gin"
)

type Dir struct {
	Dir []string `json:"dir"`
}

func main() {
	router := gin.Default()

	router.GET("/ws", controller.GinWebsocketHandler(controller.WsServer))

	router.GET("/scan", func(c *gin.Context) {
		dir := c.Query("dir")

		fmt.Println("scan dir", dir)
		config.ReadConfig()
		go controller.ScanDirs(dir)

		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "执行中",
			"data": "",
		})
	})

	router.POST("/deleteDir", func(c *gin.Context) {
		var dirs Dir
		err := c.ShouldBindJSON(&dirs)
		if err != nil {
			c.JSON(200, gin.H{
				"code": 0,
				"msg":  "执行失败，参数有误",
				"data": "",
			})
			return
		}

		fmt.Printf("%#v", dirs)
		for _, dir := range dirs.Dir {
			go func(dir string) {
				controller.DeleteDir(dir)
			}(dir)
		}

		c.JSON(200, gin.H{
			"code": 1,
			"msg":  "执行中",
			"data": "",
		})
	})

	router.GET("/send", controller.SendWs)

	router.Static("/static", "./static") //静态文件

	urlAddr := "http://127.0.0.1" + config.PORT + "/static"
	util.OpenBrowser(urlAddr)
	router.Run(config.PORT)

}
