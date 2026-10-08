package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	Type    string      `json:"type"`    // 消息类型
	Payload interface{} `json:"payload"` // 消息内容
}
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte // 待发送消息缓冲区
	UserID string      // 关联的用户 ID
}

// Hub 管理所有活跃的 WebSocket 连接
type Hub struct {
	// 所有已注册的客户端，key 为 userID
	Clients        map[string]*Client
	mu             sync.RWMutex
	OnConnected    func(userID uint) error
	OnDisconnected func(userID uint) error
	OnHeartbeat    func(userID uint) error

	// 注册/注销/广播 channel
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan []byte
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan []byte, 256),
	}
}

// Run 启动 Hub 事件循环（在独立 goroutine 中运行）
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.UserID] = client
			h.mu.Unlock()
			client.notifyConnected()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.UserID]; ok {
				delete(h.Clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
			client.notifyDisconnected()

		case message := <-h.Broadcast:
			h.mu.Lock()
			for _, client := range h.Clients {
				select {
				case client.Send <- message:
				default:

					// 缓冲区满，关闭该连接
					close(client.Send)
					delete(h.Clients, client.UserID)
				}
			}
			h.mu.Unlock()
		}
	}
}

// SendToUser 向指定用户发送消息
// data: 要发送的消息数据（已序列化为 JSON 字符串）
// 返回错误：如果用户未连接或发送失败
func (h *Hub) SendToUser(userID string, data []byte) error {

	h.mu.RLock()
	client, ok := h.Clients[userID]
	h.mu.RUnlock()

	if ok {
		select {
		case client.Send <- data:
		default:
			// 发送失败，连接可能已断开
		}
	}
	return nil
}

// IsConnected 判断用户是否存在活动的 WebSocket 连接。
func (h *Hub) IsConnected(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.Clients[userID]
	return ok
}
