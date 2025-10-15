package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
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

// 广播消息
func (s *Server) Broadcast(user *User, msg string) {
	// 将用户的消息发送到全体用户
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	s.Message <- sendMsg
}

// 事件处理
func (s *Server) Handler(conn net.Conn) {
	//用户上线
	user := NewUser(conn, s)
	user.Online()
	//监听用户是否活跃
	isLive := make(chan bool)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)

			if n == 0 {
				user.Offline()
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("conn.Read err:", err)
				return
			}
			msg := string(buf[:n-2]) //需要修改

			user.DoMessage(msg)

			isLive <- true
		}
	}()

	select {
	case <-isLive:
		// 用户活跃
		//重置定时器
	case <-time.After(time.Second * 60):
		// 用户不活跃
		user.SendMsg("time out")

		close(user.C)

		conn.Close() //关闭连接

		return //runtime.Goexit()
		//user.Offline()
	}
}

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
