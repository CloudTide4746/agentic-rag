package model

// ValidTransitions key:当前状态 value:允许的下一个状态列表
var ValidTransitions = map[TaskStatus][]TaskStatus{
	TaskStatusTodo:       {TaskStatusInProgress},
	TaskStatusInProgress: {TaskStatusReview, TaskStatusTodo},
	TaskStatusReview:     {TaskStatusDone, TaskStatusInProgress},
	TaskStatusDone:       {TaskStatusReview},
}

// CanTransition 检查是否可以从当前状态转换到下一个状态
func CanTransition(from, to TaskStatus) bool {
	allowed, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
