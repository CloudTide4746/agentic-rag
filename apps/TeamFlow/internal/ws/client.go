package ws

import (
	"fmt"
	"teamflow/pkg/utils"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second // 写超时
	pongWait       = 60 * time.Second // 等待 Pong 的超时
	pingPeriod     = 30 * time.Second // 发送 Ping 的间隔（< pongWait）
	maxMessageSize = 512              // 最大消息大小
)

// readPump 读取消息，同时处理 Pong
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))

	// 收到 Pong 时刷新 deadline 并更新在线状态
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		userID, err := utils.StringToUint(c.UserID)
		if err != nil {
			return fmt.Errorf("解析用户ID失败: %w", err)
		}
		if c.Hub.OnHeartbeat != nil {
			_ = c.Hub.OnHeartbeat(userID)
		}
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump 写消息，定时发 Ping
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) notifyConnected() {

	if c.Hub.OnConnected == nil {
		return
	}

	userID, err := utils.StringToUint(c.UserID)

	if err == nil {
		_ = c.Hub.OnConnected(userID)
	}

}

func (c *Client) notifyDisconnected() {
	if c.Hub.OnDisconnected == nil {
		return
	}
	userID, err := utils.StringToUint(c.UserID)
	if err == nil {
		_ = c.Hub.OnDisconnected(userID)
	}
}

// 这是整段代码最精彩的地方，两个 goroutine 通过 Ping/Pong 形成了一个自我维持的存活闭环：

//   writePump                         readPump
//   ────────                         ────────
//   每30s: 发 Ping  ──────────────►  收到 Pong
//                                      │
//                                      ├─ SetReadDeadline(now+60s)  刷新超时
//                                      └─ Heartbeat()              更新在线

//   1. writePump 每 30 秒发一个 Ping。
//   2. 对端（浏览器）自动回一个 Pong。
//   3. readPump 的 SetPongHandler 收到 Pong → 把 60 秒超时往后推 60 秒，并更新在线状态。
//   4. 只要 Pong 能持续回来，ReadDeadline 永远被刷新，连接永不超时，永远在线。
//   5. 一旦对端死亡（断网/崩溃），Pong 不再回来 → 60 秒后 ReadMessage 超时 → break → defer
//   里注销 + 关连接。

//   而写方向的超时（writeWait 10s）、读方向的消息大小（512B）、以及 sender 的 channel close
//   通知，共同构成了对双方资源的保护。
