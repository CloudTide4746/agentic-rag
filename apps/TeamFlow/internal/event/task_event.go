package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TaskAssignedPayload struct {
	TaskID     uint `json:"task_id"`
	ProjectID  uint `json:"project_id"`
	AssigneeID uint `json:"assignee_id"`
	OperatorID uint `json:"operator_id"`
}

func NewTaskAssigned(taskID, projectID, assigneeID, operatorID uint) Envelope {
	payload, _ := json.Marshal(TaskAssignedPayload{TaskID: taskID, ProjectID: projectID, AssigneeID: assigneeID, OperatorID: operatorID})
	now := time.Now().UTC()
	return Envelope{EventID: uuid.NewString(), EventType: TaskAssignedV1, SchemaVersion: "1", CreatedAt: now, OccurredAt: now, Payload: payload}
}
