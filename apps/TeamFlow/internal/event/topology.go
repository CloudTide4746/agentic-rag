package event

import (
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

const (
	EventsExchange      = "teamflow.events.v1"
	RetryExchange       = "teamflow.retry.v1"
	ParkingExchange     = "teamflow.parking.v1"
	NotificationQueue   = "teamflow.notification.websocket.q"
	Retry5sQueue        = "teamflow.notification.retry.5s.q"
	Retry30sQueue       = "teamflow.notification.retry.30s.q"
	Retry5mQueue        = "teamflow.notification.retry.5m.q"
	NotificationParking = "teamflow.notification.parking.q"
	TaskAssignedRouting = TaskAssignedV1
	Retry5sBinding      = "notification.5s"
	Retry30sBinding     = "notification.30s"
	Retry5mBinding      = "notification.5m"
	ParkingRouting      = "notification.parking"
)

// DeclareTopology 声明事件拓扑结构，输入为 RabbitMQ 通道，将输入的通道绑定到事件交换机、队列、绑定键等
func DeclareTopology(ch *amqp091.Channel) error {
	if ch == nil {
		return fmt.Errorf("channel is required")
	}
	if err := ch.ExchangeDeclare(EventsExchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(RetryExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare(ParkingExchange, "direct", true, false, false, false, nil); err != nil {
		return err
	}
	mainArgs := amqp091.Table{"x-dead-letter-exchange": ParkingExchange, "x-dead-letter-routing-key": ParkingRouting}
	if _, err := ch.QueueDeclare(NotificationQueue, true, false, false, false, mainArgs); err != nil {
		return err
	}
	if err := ch.QueueBind(NotificationQueue, TaskAssignedRouting, EventsExchange, false, nil); err != nil {
		return err
	}
	for _, q := range []struct {
		name, key string
		ttl       int
	}{
		{Retry5sQueue, Retry5sBinding, 5000}, {Retry30sQueue, Retry30sBinding, 30000}, {Retry5mQueue, Retry5mBinding, 300000},
	} {
		args := amqp091.Table{"x-message-ttl": q.ttl, "x-dead-letter-exchange": EventsExchange, "x-dead-letter-routing-key": TaskAssignedRouting}
		if _, err := ch.QueueDeclare(q.name, true, false, false, false, args); err != nil {
			return err
		}
		if err := ch.QueueBind(q.name, q.key, RetryExchange, false, nil); err != nil {
			return err
		}
	}
	if _, err := ch.QueueDeclare(NotificationParking, true, false, false, false, nil); err != nil {
		return err
	}
	return ch.QueueBind(NotificationParking, ParkingRouting, ParkingExchange, false, nil)
}

// DelareTopology preserves the original misspelled API.
func DelareTopology(ch *amqp091.Channel) error { return DeclareTopology(ch) }
