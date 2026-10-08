package mq

import (
	"fmt"
	"teamflow/internal/event"

	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn     *amqp091.Connection
	channels map[string]*amqp091.Channel
}
type Envelop = event.Envelope

// initMQ 初始化 RabbitMQ 连接
func initMQ() *amqp091.Connection {
	// 连接到 RabbitMQ 服务器
	conn, err := amqp091.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		panic(err)
	}
	return conn
}

// NewRabbitMQ 创建 RabbitMQ 实例
func NewRabbitMQ() *RabbitMQ {
	conn := initMQ()
	return &RabbitMQ{
		conn:     conn,
		channels: make(map[string]*amqp091.Channel),
	}
}

// GetChannel 获取 RabbitMQ 通道
func (rmq *RabbitMQ) GetChannel(name string) (*amqp091.Channel, error) {
	if ch, ok := rmq.channels[name]; ok {
		return ch, nil
	}
	return nil, fmt.Errorf("通道不存在: %s", name)
}

// NewChannel 创建 RabbitMQ 通道
func (rmq *RabbitMQ) NewChannel(name string) (*amqp091.Channel, error) {
	if _, ok := rmq.channels[name]; ok {
		return nil, fmt.Errorf("通道已存在: %s", name)
	}
	ch, err := rmq.conn.Channel()
	if err != nil {
		return nil, err
	}
	rmq.channels[name] = ch
	return ch, nil
}

// NewQueue 创建 RabbitMQ 队列
func (rmq *RabbitMQ) NewQueue(name string, channel string) amqp091.Queue {
	ch, err := rmq.NewChannel(channel)
	if err != nil {
		panic(err)
	}
	queue, err := ch.QueueDeclare(
		name,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	)

	if err != nil {
		panic(err)
	}
	return queue
}

// CloseConnection 关闭 RabbitMQ 连接
func CloseConnection(conn *amqp091.Connection) {
	conn.Close()
}

// NewMessage 创建 RabbitMQ 消息
func NewMessage(body []byte) amqp091.Publishing {
	return amqp091.Publishing{
		Body: body,
	}
}

// Publish 发布 RabbitMQ 消息
//func (rmq *RabbitMQ) Publish(exchange, routingKey string, msg amqp091.Publishing) {
//	rmq.channels[channel].Publish(exchange, routingKey, false, false, msg)
//}

// Consume 消费 RabbitMQ 消息
func (rmq *RabbitMQ) Consume(_ Envelop, channelName string, queue *amqp091.Queue, autoAck bool) (<-chan amqp091.Delivery, error) {
	if queue == nil || queue.Name == "" {
		return nil, fmt.Errorf("queue is required")
	}
	ch, err := rmq.GetChannel(channelName)
	if err != nil {
		return nil, err
	}
	deliveries, err := ch.Consume(
		queue.Name,
		"",
		autoAck,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return deliveries, nil
}

// SetQos 设置 RabbitMQ QoS
func (rmq *RabbitMQ) SetQos(channelName string, prefetchCount uint16, global bool) error {
	ch, err := rmq.GetChannel(channelName)
	if err != nil {
		return err
	}
	return ch.Qos(int(prefetchCount), 0, global)
}
