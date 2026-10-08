package controller

// APIResponse documents the common JSON envelope returned by the API.
type APIResponse struct {
	Code      int         `json:"code" example:"0"`
	Message   string      `json:"message" example:"success"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp" example:"1780000000"`
}
