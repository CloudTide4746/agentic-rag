package dto

type PageResult[T any] struct {
	Total int64 `json:"total"` // 总记录数
	Page  int   `json:"page"`  // 当前页码（从 1 开始）
	Size  int   `json:"size"`  // 每页条数
	List  []T   `json:"list"`  // 当前页数据列表
}

// TotalPages 计算总页数
func (r *PageResult[T]) TotalPages() int {
	if r.Size == 0 {
		return 0
	}
	return int((r.Total + int64(r.Size) - 1) / int64(r.Size))
}
