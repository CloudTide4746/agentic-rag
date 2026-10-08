package pagination

import "gorm.io/gorm"

type Pagination struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
}

// Normalize 归一化分页参数
func (p *Pagination) Normalize() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
}

func (p *Pagination) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

// Paginate 返回 GORM Scope 函数，用于链式调用
// 用法：db.Scopes(pagination.Paginate(page, size)).Find(&tasks)
func Paginate(page, size int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		offset := (page - 1) * size
		return db.Offset(offset).Limit(size)
	}
}
