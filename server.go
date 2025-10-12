package main

import (
	"fmt"
	"io"
	"net"
	"sync"
)

type Server struct {
	Ip   string
	Port int

	//在线用户列表
	OnlineMap map[string]*User
	//保护在线用户列表的锁
	mapLock sync.RWMutex

	//消息广播的channel
	Message chan string
}

// 创建一个接口
func NewServer(ip string, port int) *Server {
	return &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
}

// 监听广播消息发送给所有在线用户
func (s *Server) ListenMessager() {
	for {
		msg := <-s.Message
		//将msg发送给所有的在线用户
		s.mapLock.Lock()
		for _, user := range s.OnlineMap {
			user.C <- msg
		}
		s.mapLock.Unlock()
	}
}
func (s *Server) Broadcast(user *User, msg string) {
	// 将用户的消息发送到全体用户
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	s.Message <- sendMsg
}

func (s *Server) Handler(conn net.Conn) {

	//当前连接的业务
	fmt.Println("连接建立成功")

	//用户上线
	s.mapLock.Lock()
	user := NewUser(conn)
	s.OnlineMap[user.Name] = user
	s.mapLock.Unlock()
	//广播当前用户上线的消息
	s.Broadcast(user, "已上线")
	//接受客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)

			if n == 0 {
				s.Broadcast(user, "下线")
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("conn.Read err:", err)
				return
			}
			msg := string(buf[:n-1])

			s.Broadcast(user, msg)
		}
	}()

	select {}
}

// 广播消息

func (s *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		println("net.Listen error:", err)
		return
	}
	//关闭监听socket
	defer listener.Close()

	//启动监听Message的goroutine
	go s.ListenMessager()
	for {
		// accept
		conn, err := listener.Accept()
		if err != nil {
			println("listener.Accept error:", err)
			continue
		}
		// do handler
		go s.Handler(conn)
	}
}
