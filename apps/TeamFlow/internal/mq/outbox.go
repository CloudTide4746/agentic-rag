package mq

import (
	"context"
	"errors"
	"fmt"
	"teamflow/internal/model"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxOutboxAttempts = 5

func retryDelay(attempt int) time.Duration {
	switch attempt {
	case 0:
		return time.Second
	case 1:
		return time.Second * 2
	case 2:
		return time.Second * 8
	case 3:
		return time.Second * 30
	case 4:
		return time.Minute * 2
	default:
		return time.Minute * 5
	}
}

// 此处让publisher里面的Publish实现OutboxPublisher接口，用于发布消息到指定 exchange 的指定路由键
type OutboxPublisher interface {
	Publish(ctx context.Context, exchange string, routingKey string, body []byte, headers amqp091.Table) error
}

// OutboxRelay 是一个消息中继器，接入publisher，用于将消息发布到指定 exchange 的指定路由键，同时记录消息到数据库
// 消息记录到数据库的字段包括消息ID、消息内容、消息头、创建时间、更新时间
// RelayID 用于标识当前消息中继器的实例，确保消息只被一个实例处理一次
type OutboxRelay struct {
	db        *gorm.DB
	publisher OutboxPublisher
	relayID   string
}

// NewOutboxRelay 创建一个消息中继器实例
func NewOutboxRelay(db *gorm.DB, publisher OutboxPublisher, relayID string) *OutboxRelay {
	return &OutboxRelay{
		db:        db,
		publisher: publisher,
		relayID:   relayID,
	}
}

// ClaimNext 从数据库中获取下一个待处理的消息，开始处理该消息
func (r *OutboxRelay) claimNext(ctx context.Context, now time.Time) (*model.OutboxEvent, error) {
	var outboxEvent model.OutboxEvent
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{
			Strength: "UPDATE",
			Options:  "SKIP LOCKED",
		}).Where("status = ? AND next_attempt_at <= ?", model.OutboxPending, now).Order("next_attempt_at ASC, id ASC").First(&outboxEvent).Error
		if err != nil {
			return err
		}
		result := tx.Model(&model.OutboxEvent{}).
			Where(
				"id = ? AND status = ?",
				outboxEvent.ID,
				model.OutboxPending,
			).
			Updates(map[string]interface{}{
				"status":    model.OutboxPublishing,
				"locked_by": r.relayID,
				"locked_at": now,
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != 1 {
			return fmt.Errorf(
				"outbox event %d was not claimed",
				outboxEvent.ID,
			)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	outboxEvent.Status = model.OutboxPublishing
	outboxEvent.LockedBy = &r.relayID
	outboxEvent.LockedAt = &now
	return &outboxEvent, nil
}

// MarkAsPublished 标记消息为已发布
// 此处一整条调用链的context都是同一个，上层如果去取消，下层也会取消
func (r *OutboxRelay) markAsPublished(ctx context.Context, outboxID uint, now time.Time) error {
	// 校验消息是否存在且状态为Publishing,并且必须由对应RelayID锁定
	result := r.db.WithContext(ctx).Model(&model.OutboxEvent{}).Where(
		"id = ? and status = ? AND locked_by = ?",
		outboxID,
		model.OutboxPublishing,
		r.relayID,
	).Updates(map[string]interface{}{
		"status":       model.OutboxPublished,
		"attempts":     gorm.Expr("attempts + 1"),
		"published_at": now,
		"last_error":   "",
		"locked_by":    nil,
		"locked_at":    nil,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf(
			"outbox event %d was not published",
			outboxID,
		)

	}
	return nil
}

// markPublishFailure 标记消息为失败记录
func (r *OutboxRelay) markPublishFailure(ctx context.Context, outboxID uint, attempt int, err error) error {
	var status model.OutboxStatus
	nextAttemptAt := time.Now().UTC().Add(retryDelay(attempt))
	if attempt >= maxOutboxAttempts {
		status = model.OutboxFailed
		nextAttemptAt = time.Now().UTC()
	} else {
		status = model.OutboxPending
	}

	result := r.db.WithContext(ctx).Model(&model.OutboxEvent{}).Where(
		"id = ? and status = ? AND locked_by = ?",
		outboxID,
		model.OutboxPublishing,
		r.relayID,
	).Updates(map[string]interface{}{
		"status":          status,
		"attempts":        attempt,
		"next_attempt_at": nextAttemptAt,
		"last_error":      err.Error(),
		"locked_by":       nil,
		"locked_at":       nil,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf(
			"outbox event %d was not marked as failed",
			outboxID,
		)
	}
	return nil
}

// DispatchOnce 处理一个消息
func (r *OutboxRelay) DispatchOnce(
	ctx context.Context,
) (bool, error) {
	claimedAt := time.Now().UTC()

	outboxEvent, err := r.claimNext(ctx, claimedAt)

	// 没有待发送消息属于正常情况。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	// 数据库查询或领取失败。
	if err != nil {
		return false, fmt.Errorf(
			"claim outbox event: %w",
			err,
		)
	}

	// 必须领取成功以后才能读取 outboxEvent。
	headers := amqp091.Table{
		"event_id":   outboxEvent.EventID,
		"event_type": outboxEvent.EventType,
	}

	err = r.publisher.Publish(
		ctx,
		outboxEvent.Exchange,
		outboxEvent.RoutingKey,
		outboxEvent.Payload,
		headers,
	)

	if err != nil {
		attempt := outboxEvent.Attempts + 1

		recordErr := r.markPublishFailure(
			ctx,
			outboxEvent.ID,
			attempt,
			err,
		)
		if recordErr != nil {
			return true, fmt.Errorf(
				"publish failed: %v; record failure: %w",
				err,
				recordErr,
			)
		}

		return true, fmt.Errorf(
			"publish outbox event %s: %w",
			outboxEvent.EventID,
			err,
		)
	}

	err = r.markAsPublished(
		ctx,
		outboxEvent.ID,
		time.Now().UTC(),
	)
	if err != nil {
		return true, fmt.Errorf(
			"mark outbox event %s published: %w",
			outboxEvent.EventID,
			err,
		)
	}

	return true, nil
}
