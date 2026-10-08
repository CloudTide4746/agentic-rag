package dto

type CursorPageRequest struct {
	Cursor string `json:"cursor"`
	Size   int    `json:"size"`
}

type CursorPageResult[T any] struct {
	List       []T    `json:"list"`
	NextCursor string `json:"next_cursor"`
	//空字符串表示没有更多数据
	HasMore bool `json:"has_more"`
}
