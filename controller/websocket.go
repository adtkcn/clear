package controller

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/websocket"
)

type ClientWs struct {
	uid int32
	ws  *websocket.Conn
}

var clientWsGroup []ClientWs

// gin转http中间件
func GinWebsocketHandler(wsConnHandle websocket.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("new ws request: %v", c.Request.RemoteAddr)
		if c.IsWebsocket() {
			wsConnHandle.ServeHTTP(c.Writer, c.Request)
		} else {
			_, _ = c.Writer.WriteString("===not websocket request===")
		}
	}
}

func WsServer(ws *websocket.Conn) {
	uid := int32(len(clientWsGroup))
	clientWs := ClientWs{
		uid: uid,
		ws:  ws,
	}
	clientWsGroup = append(clientWsGroup, clientWs)
	log.Printf("clientWsGroup: %v", clientWsGroup)

	for {
		var msg string
		if err := websocket.Message.Receive(ws, &msg); err != nil {
			log.Printf("接收错误 %v", err)
			break
		}

		log.Printf("接收: %v", msg)

		data := []byte(time.Now().Format(time.RFC3339))
		if _, err := ws.Write(data); err != nil {
			log.Printf("写错误 %v", err)
			break
		}
	}
	DeleteClient(uid)

}

func DeleteClient(uid int32) {
	for i := len(clientWsGroup) - 1; i >= 0; i-- {
		if uid == clientWsGroup[i].uid {
			clientWsGroup = append(clientWsGroup[:i], clientWsGroup[(i+1):]...)
		}
	}
	log.Printf("用uid删除 %v,%v", clientWsGroup, uid)
}

func BatchSendWs(typeStr string, msg string) {
	// var uid, _ = strconv.Atoi(c.Query("uid"))
	for i := 0; i < len(clientWsGroup); i++ {
		user := clientWsGroup[i]
		send := map[string]string{
			"type": typeStr,
			"data": msg,
		}
		str, _ := json.Marshal(send)
		data := []byte(str)
		if _, err := user.ws.Write(data); err != nil {
			log.Println("批量发送错误", err)
			continue
		}
	}
}

func SendWs(c *gin.Context) {
	// var uid, _ = strconv.Atoi(c.Query("uid"))
	for i := 0; i < len(clientWsGroup); i++ {
		user := clientWsGroup[i]
		data := []byte(time.Now().Format(time.RFC3339))
		if _, err := user.ws.Write(data); err != nil {
			log.Println("批量发送错误", err)
			continue
		}
	}
}
