package service

import (
	"errors"
	"teamflow/internal/cache"
	"teamflow/internal/model"
	"teamflow/internal/repository"
)

var (
	ErrCommentNotFound     = errors.New("评论不存在")
	ErrCommentForbidden    = errors.New("无权操作此评论")
	ErrCommentContentEmpty = errors.New("评论内容不能为空")
)

type CreateCommentInput struct {
	Comments []*model.Comment `json:"comments"`
	TaskID   uint             `json:"task_id"`
	UserID   uint             `json:"user_id"`
	Total    int64            `json:"total"`
	Content  string           `json:"content"`
	Page     int              `json:"page"`
	Size     int              `json:"size"`
}
type CommentListResult struct {
	Comments []*model.Comment `json:"comments"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}
type commentService struct {
	repo repository.CommentRepository
}

type commentListCacheValue struct {
	Comments []*model.Comment `json:"comments"`
	Total    int64            `json:"total"`
}

type CommentService interface {
	Create(input CreateCommentInput) (*model.Comment, error)
	GetByTaskID(taskID uint, page, size int) (*CommentListResult, error)
	Delete(commentID, operatorID uint, isAdmin bool) error
}

func NewCommentService(repo repository.CommentRepository) CommentService {
	return &commentService{repo: repo}
}

func (s *commentService) Create(input CreateCommentInput) (*model.Comment, error) {
	if input.Content == "" {
		return nil, ErrCommentContentEmpty
	}
	comment := &model.Comment{
		TaskID:  input.TaskID,
		UserID:  input.UserID,
		Content: input.Content,
	}
	if err := s.repo.Create(comment); err != nil {
		return nil, err
	}
	_ = cache.InvalidateTaskComments(input.TaskID)
	return comment, nil
}
func (s *commentService) GetByTaskID(taskID uint, page, size int) (*CommentListResult, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	cacheKey := cache.TaskCommentsKey(taskID, page, size)
	var cached commentListCacheValue
	if ok, _ := cache.GetResourceCache(cacheKey, &cached); ok {
		return &CommentListResult{Comments: cached.Comments, Total: cached.Total, Page: page, PageSize: size}, nil
	}
	comments, total, err := s.repo.FindByTaskID(taskID, page, size)
	if err != nil {
		return nil, err
	}
	result := &CommentListResult{
		Comments: comments,
		Total:    total,
		Page:     page,
		PageSize: size,
	}
	_ = cache.SetListCache(cacheKey, commentListCacheValue{Comments: comments, Total: total})
	return result, nil
}

// Delete 删除评论（含权限校验）
// operatorID：执行删除操作的用户ID；isAdmin：操作者是否为管理员
func (s *commentService) Delete(commentID, operatorID uint, isAdmin bool) error {
	comment, err := s.repo.FindByID(commentID)
	if err != nil {
		return ErrCommentNotFound
	}
	// 管理员可删除任何评论，普通用户只能删除自己的评论
	if !isAdmin && comment.UserID != operatorID {
		return ErrCommentForbidden
	}
	if err := s.repo.Delete(commentID); err != nil {
		return err
	}
	_ = cache.InvalidateTaskComments(comment.TaskID)
	return nil
}
