package publisher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	ch          *amqp091.Channel // 此 Publisher 是这个 Channel 的唯一 owner
	mu          *sync.Mutex
	confirmOnce *sync.Once
	confirmErr  error
	confirms    <-chan amqp091.Confirmation
	returns     <-chan amqp091.Return
}

func NewPublisher(ch *amqp091.Channel) *Publisher {
	return &Publisher{ch: ch, mu: &sync.Mutex{}, confirmOnce: &sync.Once{}}
}

// Publish 直接把原始字节消息发布到指定 exchange 的指定路由键
// 使用原生的PublishWithContext方法，设置DeliveryMode为Persistent，实现消息持久化
func (p *Publisher) Publish(ctx context.Context, exchange, routingKey string, body []byte, headers amqp091.Table) error {
	if p == nil || p.ch == nil {
		return errors.New("publisher channel is not configured")
	}
	if p.mu == nil || p.confirmOnce == nil {
		return errors.New("publisher is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	waitCtx := ctx
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		waitCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	// 上锁，确保只有一个 Publisher 能用 Confirm 方法，避免并发调用导致的错误
	p.mu.Lock()
	defer p.mu.Unlock()
	p.confirmOnce.Do(func() {
		p.confirmErr = p.ch.Confirm(false)
		if p.confirmErr == nil {
			p.confirms = p.ch.NotifyPublish(make(chan amqp091.Confirmation, 1))
			p.returns = p.ch.NotifyReturn(make(chan amqp091.Return, 1))
		}
	})
	if p.confirmErr != nil {
		return fmt.Errorf("enable confirm publish %s/%s: %w", exchange, routingKey, p.confirmErr)
	}
	if err := p.ch.PublishWithContext(ctx, exchange, routingKey, true, false, amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent,

		Headers: headers,
		Body:    body,
	}); err != nil {
		return fmt.Errorf("publish %s/%s: %w", exchange, routingKey, err)
	}
	// 等待确认或返回消息 Confirmation
	select {
	case returned := <-p.returns:
		return fmt.Errorf("publish %s/%s returned by broker: %s", exchange, routingKey, returned.ReplyText)
	case confirmation := <-p.confirms:
		if !confirmation.Ack {
			return fmt.Errorf("publish %s/%s was rejected by broker", exchange, routingKey)
		}
		return nil
	case <-waitCtx.Done():
		return fmt.Errorf("publish %s/%s confirmation: %w", exchange, routingKey, waitCtx.Err())
	}
}

// 普通 PublishWithContext 返回 nil，主要表示客户端成功把帧写入了连接；
// 它不等于 Broker 已经把消息接管并落到目标 Queue。连接可能在写入后立即断开，
// Publisher 无法仅靠函数返回值判断最终结果。
