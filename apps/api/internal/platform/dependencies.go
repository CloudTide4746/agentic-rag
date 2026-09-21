package platform

import (
	"github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Dependencies declares the future composition boundary. No connection is
// created until a feature explicitly wires its dependencies at the entry point.
type Dependencies struct {
	DB    *gorm.DB
	Redis *redis.Client
	Queue *amqp091.Connection
}

// MySQLDialector must only receive the platform database DSN, never Agent data.
func MySQLDialector(dsn string) gorm.Dialector { return mysql.Open(dsn) }
