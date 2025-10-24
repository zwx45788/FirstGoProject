// user.go
package main

import (
	"strings"

	"github.com/gorilla/websocket"
)

type User struct {
	Name   string
	C      chan string
	conn   *websocket.Conn // 关键改动：从 net.Conn 改为 websocket.Conn
	server *Server
}

// 创建一个用户的API
func NewUser(conn *websocket.Conn, server *Server) *User {
	// 使用连接的远程地址作为初始用户名
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}

	// 启动监听当前user channel消息的goroutine
	go user.ListenMessage()

	return user
}

// 用户上线业务
func (u *User) Online() {
	u.server.mapLock.Lock()
	u.server.OnlineMap[u.Name] = u
	u.server.mapLock.Unlock()
	u.server.Broadcast(u, "is online")
}

// 用户下线业务
func (u *User) Offline() {
	u.server.mapLock.Lock()
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock()
	u.server.Broadcast(u, "is offline")
}

// 给当前用户对应的客户端发送消息
func (u *User) SendMsg(msg string) {
	u.conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// 处理用户消息
func (u *User) DoMessage(msg string) {
	if msg == "who" {
		u.server.mapLock.Lock()
		for _, user := range u.server.OnlineMap {
			onlineMsg := "[" + user.Name + "]" + ":online..."
			u.C <- onlineMsg
		}
		u.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]
		u.server.mapLock.Lock()
		if _, ok := u.server.OnlineMap[newName]; ok {
			u.SendMsg("the name already exists")
		} else {
			delete(u.server.OnlineMap, u.Name)
			u.server.OnlineMap[newName] = u
			u.Name = newName
			u.SendMsg("you have changed your name to:" + u.Name)
		}
		u.server.mapLock.Unlock()
	} else if len(msg) > 4 && msg[:3] == "to|" {
		parts := strings.SplitN(msg, "|", 3)
		if len(parts) < 3 {
			u.SendMsg("please use correct format: to|username|message content")
			return
		}
		u.server.mapLock.Lock()
		defer u.server.mapLock.Unlock()
		remoteName := parts[1]
		content := parts[2]
		if remoteUser, ok := u.server.OnlineMap[remoteName]; ok {
			remoteUser.SendMsg(u.Name + " to you:" + content)
		} else {
			u.SendMsg("the user does not exist")
		}
	} else {
		u.server.Broadcast(u, msg)
	}
}

// 监听当前User channel的方法，一旦有消息，就直接发送给客户端
func (u *User) ListenMessage() {
	for msg := range u.C {
		u.SendMsg(msg)
	}
}
