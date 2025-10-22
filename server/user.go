package main

import (
	"net"
	"strings"
)

type User struct {
	Addr   string
	Name   string
	C      chan string
	conn   net.Conn
	server *Server
}

// 创建用户的api
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Addr:   userAddr,
		Name:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}
	//启动监听当前user channel消息的goroutine
	go user.ListenMessage()

	return user
}

// 用户上线业务
func (u *User) Online() {
	u.server.mapLock.Lock()
	u.server.OnlineMap[u.Name] = u
	u.server.mapLock.Unlock()
	//广播当前用户上线的消息
	u.server.Broadcast(u, "is online")
}

// 用户下线业务
func (u *User) Offline() {
	u.server.mapLock.Lock()
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock()
	//广播当前用户下线的消息
	u.server.Broadcast(u, "is offline")

}

// 给当前user对应的客户端发消息
func (u *User) SendMsg(msg string) {
	u.conn.Write([]byte(msg))
}
func (u *User) DoMessage(msg string) {
	if msg == "who" {
		//查询当前在线用户都有哪些
		u.server.mapLock.Lock()
		for _, user := range u.server.OnlineMap {
			onlineMsg := "[" + user.Addr + "]" + user.Name + ":online..."
			u.C <- onlineMsg
		}
		u.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		// 更改用户名 消息格式：rename|张三
		newName := msg[7:]
		_, ok := u.server.OnlineMap[newName]
		if ok {
			u.SendMsg("the name already exists\n")
		} else {
			u.server.mapLock.Lock()
			delete(u.server.OnlineMap, u.Name)
			u.server.OnlineMap[newName] = u
			u.server.mapLock.Unlock()
			u.Name = newName
			u.SendMsg("you have changed your name to:" + u.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 私聊 消息格式：to|张三|消息内容
		parts := strings.SplitN(msg, "|", 3) // 使用 SplitN，最多只分成3份

		// 1. 检查格式是否正确：必须是 "to|名字|内容" 三部分
		if len(parts) < 3 {
			u.SendMsg("please use correct format: to|username|message content\n")
			return
		}
		u.server.mapLock.Lock()
		defer u.server.mapLock.Unlock()
		// 2. 提取用户名和消息内容
		remoteName := parts[1]
		content := parts[2]
		remoteUser, ok := u.server.OnlineMap[remoteName]
		if !ok {
			u.SendMsg("the user does not exist\n")
			return
		}
		if content == "" {
			u.SendMsg("the message is empty\n")
			return
		}
		remoteUser.SendMsg(u.Name + " to you:" + content + "\n")
	} else {
		println(msg)
		u.server.Broadcast(u, msg)
	}
}

// 监听当前User channel的方法，一旦有消息，就直接发送给客户端
func (u *User) ListenMessage() {
	// 使用 for range 遍历 channel
	for msg := range u.C {
		u.conn.Write([]byte(msg + "\n"))
	}
	// 当 for range 循环正常结束时，说明 u.C 已经被关闭
	// 这个 goroutine 的使命就完成了，它自然就结束了

	//这样写会有问题，当用户下线时，goroutine不会结束，导致内存泄漏
	// for {
	//     msg := <-u.C // 从 channel 里读取
	//     u.conn.Write([]byte(msg + "\n"))
	// }
}
