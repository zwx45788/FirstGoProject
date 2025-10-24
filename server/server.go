// server.go
package main

import (
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
		Message:   make(chan string),
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

		// 监听用户消息
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
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	// 启动消息广播的goroutine
	go s.ListenMessager()

	// 启动Web服务器
	fmt.Println("🚀 服务器启动成功！")
	fmt.Println("🌐 请访问: http://localhost:8081")
	fmt.Println("💬 WebSocket地址: ws://localhost:8081/ws")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", s.Port), nil))
}

// 监听广播消息channel，一旦有消息就发送给所有在线用户
func (s *Server) ListenMessager() {
	for {
		msg := <-s.Message
		s.mapLock.Lock()
		for _, user := range s.OnlineMap {
			user.C <- msg
		}
		s.mapLock.Unlock()
	}
}

// 广播消息的方法
func (s *Server) Broadcast(user *User, msg string) {
	sendMsg := "[" + user.Name + "]:" + msg
	s.Message <- sendMsg
}
