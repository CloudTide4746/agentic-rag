package main

import "log"

// main 报告 Worker 尚未实现并退出，避免将不可用的消息处理能力视为启动成功。
func main() {

	log.Fatal("worker is not implemented; message processing is unavailable")
}
