package main

import "go.uber.org/zap"

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()
	// Exit explicitly rather than silently pretending to consume reliable jobs.
	log.Warn("worker_not_implemented", zap.String("message", "Outbox、队列消费与 Python 幂等适配尚未接入；此入口仅用于编译验证"))
}
