package chat

import "context"


type CreateChatReq struct {

}
type CreateChatResp struct {

}
type GetChatReq struct {

}
type GetChatResp struct {

}
type SendMessageReq struct {

}
type SendMessageResp struct {

}
type GetMessagesReq struct {

}
type GetMessagesResp struct {

}
type chatHandler struct {
}
type ChatHandler interface {
	CreateChat(ctx context.Context, req *CreateChatReq) (*CreateChatResp, error) //POST /api/chat/sessions
	GetChat(ctx context.Context, req *GetChatReq) (*GetChatResp, error) //GET /api/chat/sessions/{session_id}/messages
	SendMessage(ctx context.Context, req *SendMessageReq) (*SendMessageResp, error)  //POST /api/chat/sessions/{session_id}/ask
	GetMessages(ctx context.Context, req *GetMessagesReq) (*GetMessagesResp, error) //GET /api/chat/sessions
}
