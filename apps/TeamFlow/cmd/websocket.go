package main

import (
	"net/http"
	"teamflow/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func SetupWebSocketRoutes(r *gin.Engine, hub *ws.Hub) {
	r.GET("/ws", func(c *gin.Context) {
		ServeWs(hub, c.Writer, c.Request)
	})
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer conn.Close()
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = conn.WriteMessage(messageType, message)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

}

// ServeWs 处理单个 WebSocket 连接
func ServeWs(hub *ws.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// 从 JWT 中解析用户 ID（实际项目中需验证 token）
	userID := r.URL.Query().Get("user_id")

	// 创建客户端并注册到 Hub
	client := &ws.Client{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
	}

	// 启动读写 goroutine
	hub.Register <- client
	go client.WritePump()
	go client.ReadPump()
}
