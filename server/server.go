// server.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Server struct {
	Ip        string
	Port      int
	OnlineMap map[string]*User
	mapLock   sync.RWMutex
	Message   chan string
}

// 创建一个Server接口
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string, 100),
	}
	return server
}

// 启动服务器的方法
func (s *Server) Start() {
	// 创建WebSocket升级器
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有来源
		},
	}

	// 处理WebSocket连接的路由
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket升级失败: %v", err)
			return
		}

		user := NewUser(conn, s)
		user.Online()

		// 监听用户消息写入管道
		go func() {
			for {
				_, p, err := conn.ReadMessage()
				if err != nil {
					user.Offline()
					return
				}
				user.DoMessage(string(p))
			}
		}()
	})

	// 提供静态文件服务 (HTML页面)
	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	http.ServeFile(w, r, "index.html")
	// })

	// 假设你的前端文件放在 ./public 目录（与 server 可相对或绝对路径）
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs) // 所有静态资源（index.html、style.css、script.js）都会被正确返回

	// 启动消息广播的goroutine
	go s.ListenMessager()

	// 启动Web服务器
	fmt.Println("🚀 服务器启动成功！")
	fmt.Println("🌐 请访问: http://localhost:8081")
	fmt.Println("💬 WebSocket地址: ws://localhost:8081/ws")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil))
}

// 将管道中的消息分发给所有用户
func (s *Server) ListenMessager() {
	for msg := range s.Message {
		s.mapLock.RLock()
		for _, user := range s.OnlineMap {
			select {
			case user.C <- msg:
			default:
				log.Printf("! client %s buffer full, message discarded\n", user.Name)
			}
		}
		s.mapLock.RUnlock()
	}
}

// 将用户发送的信息写入管道
func (s *Server) Broadcast(msg Message) {
	data, _ := json.Marshal(msg)
	s.Message <- string(data)
}
