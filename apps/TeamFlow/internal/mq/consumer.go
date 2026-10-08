package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"teamflow/internal/event"
	"teamflow/internal/publisher"

	"github.com/rabbitmq/amqp091-go"
)

var ErrPermanent = event.ErrPermanent

type Consumer struct {
	rmq       *RabbitMQ
	publisher *publisher.Publisher
	handler   event.Handler
	registry  *Registry
}

// NewConsumer creates a consumer. The optional publisher keeps compatibility
// with the original two-argument constructor while allowing retry/parking.
func NewConsumer(rmq *RabbitMQ, handler *event.EventHandler, publishers ...*publisher.Publisher) *Consumer {
	var pub *publisher.Publisher
	if len(publishers) > 0 {
		pub = publishers[0]
	}
	c := &Consumer{rmq: rmq, publisher: pub, handler: handler, registry: NewRegistry()}
	if handler != nil {
		_ = c.registry.Register(event.TaskAssignedV1, handler)
	}
	return c
}

func Handle(envelop Envelop) error {
	handler, err := Get(envelop.EventType)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrPermanent, err)
	}
	return handler.Handle(envelop)
}

func (c *Consumer) handle(envelop Envelop) error {
	if c != nil && c.registry != nil {
		if handler, err := c.registry.Get(envelop.EventType); err == nil {
			return handler.Handle(envelop)
		}
	}
	return Handle(envelop)
}

func IspermenantError(err error) bool { return errors.Is(err, ErrPermanent) }

func IsPermanentError(err error) bool { return errors.Is(err, ErrPermanent) }

func (c *Consumer) Park(delivery amqp091.Delivery, cause error) error {
	if c == nil || c.publisher == nil {
		return errors.New("publisher is not configured")
	}
	headers := cloneHeaders(delivery.Headers)
	if cause != nil {
		headers["x-park-reason"] = cause.Error()
	}
	if err := c.publisher.Publish(context.Background(), event.ParkingExchange, event.ParkingRouting, delivery.Body, headers); err != nil {
		return err
	}
	return delivery.Ack(false)
}

func (c *Consumer) Consumer_Start(envelop Envelop, channelName string, queue *amqp091.Queue) error {
	if c == nil || c.rmq == nil {
		return errors.New("rabbitmq is not configured")
	}
	deliveries, err := c.rmq.Consume(envelop, channelName, queue, false)
	if err != nil {
		return err
	}
	for delivery := range deliveries {
		var current Envelop
		if err := json.Unmarshal(delivery.Body, &current); err != nil || current.EventType == "" {
			if parkErr := c.Park(delivery, fmt.Errorf("%w: decode event: %v", ErrPermanent, err)); parkErr != nil {
				_ = delivery.Nack(false, true)
			}
			continue
		}

		if err := c.handle(current); err == nil {
			if ackErr := delivery.Ack(false); ackErr != nil {
				return ackErr
			}
			continue
		} else if IspermenantError(err) {
			if parkErr := c.Park(delivery, err); parkErr != nil {
				_ = delivery.Nack(false, true)
			}
			continue
		} else if retryErr := c.Retry(delivery, err); retryErr != nil {
			if IspermenantError(retryErr) {
				if parkErr := c.Park(delivery, retryErr); parkErr != nil {
					_ = delivery.Nack(false, true)
				}
			} else {
				_ = delivery.Nack(false, true)
			}
			continue
		}
		if ackErr := delivery.Ack(false); ackErr != nil {
			return ackErr
		}
	}
	return nil
}

func (c *Consumer) Retry(delivery amqp091.Delivery, cause error) error {
	attempt := retryCount(delivery.Headers) + 1
	var routingKey string
	switch attempt {
	case 1:
		routingKey = event.Retry5sBinding
	case 2:
		routingKey = event.Retry30sBinding
	case 3:
		routingKey = event.Retry5mBinding
	default:
		return fmt.Errorf("%w: retry limit exceeded", ErrPermanent)
	}
	if c == nil || c.publisher == nil {
		return errors.New("publisher is not configured")
	}
	headers := cloneHeaders(delivery.Headers)
	headers["x-retry-count"] = int32(attempt)
	// Keep the legacy header name so messages already in flight continue to retry correctly.
	headers["retry_count"] = int64(attempt)
	if cause != nil {
		headers["x-retry-reason"] = cause.Error()
	}
	return c.publisher.Publish(context.Background(), event.RetryExchange, routingKey, delivery.Body, headers)
}

func cloneHeaders(headers amqp091.Table) amqp091.Table {
	result := make(amqp091.Table, len(headers)+1)
	for key, value := range headers {
		result[key] = value
	}
	return result
}

func retryCount(headers amqp091.Table) int32 {
	for _, key := range []string{"x-retry-count", "retry_count"} {
		if value, ok := headers[key]; ok {
			switch count := value.(type) {
			case int:
				return int32(count)
			case int8:
				return int32(count)
			case int16:
				return int32(count)
			case int32:
				return count
			case int64:
				return int32(count)
			case uint:
				return int32(count)
			case uint32:
				return int32(count)
			case uint64:
				return int32(count)
			}
		}
	}
	return 0
}
