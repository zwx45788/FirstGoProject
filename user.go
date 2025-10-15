package main

import "net"

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

//用户上线业务
func (u *User) Online() {
	u.server.mapLock.Lock()
	u.server.OnlineMap[u.Name] = u
	u.server.mapLock.Unlock()
	//广播当前用户上线的消息
	u.server.Broadcast(u, "is online")
}

//用户下线业务
func (u *User) Offline() {
	u.server.mapLock.Lock()
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock()
	//广播当前用户下线的消息
	u.server.Broadcast(u, "is offline")

}

//给当前user对应的客户端发消息
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
	} else {
		println(msg)
		u.server.Broadcast(u, msg)
	}
}

// 监听当前User channel的方法，一旦有消息，就直接发送给客户端
func (u *User) ListenMessage() {
	for {
		msg := <-u.C
		u.conn.Write([]byte(msg + "\n"))
	}
}
