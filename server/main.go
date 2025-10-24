// main.go
package main

import "flag"

func main() {
	// 解析命令行参数
	port := flag.Int("port", 8081, "设置服务器端口")
	ip := flag.String("ip", "127.0.0.1", "设置服务器IP地址")
	flag.Parse()

	// 创建服务器实例
	server := NewServer(*ip, *port)

	// 启动服务器
	server.Start()
}
