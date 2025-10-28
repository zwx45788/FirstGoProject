// user.go
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type User struct {
	Name   string
	C      chan string
	conn   *websocket.Conn // 关键改动：从 net.Conn 改为 websocket.Conn
	server *Server
}

type Message struct {
	Type    string   `json:"type"`
	From    string   `json:"from,omitempty"`
	To      string   `json:"to,omitempty"`
	Content string   `json:"content,omitempty"`
	NewName string   `json:"newName,omitempty"` // 前端期望的字段
	Users   []string `json:"users,omitempty"`   // 用于返回用户列表
}

// 创建一个用户的API
func NewUser(conn *websocket.Conn, server *Server) *User {
	// 使用连接的远程地址作为初始用户名
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name:   userAddr,
		C:      make(chan string, 16),
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

	// 构造一条系统消息
	msg := Message{
		Type:    "system",
		From:    "server",
		Content: fmt.Sprintf("%s is online", u.Name),
	}

	// 广播 JSON 消息
	u.server.Broadcast(msg)
}

// 用户下线业务
func (u *User) Offline() {
	u.server.mapLock.Lock()
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock()
	msg := Message{
		Type:    "system",
		From:    "server",
		Content: fmt.Sprintf("%s is offline", u.Name),
	}

	// 广播 JSON 消息
	u.server.Broadcast(msg)
}

// 给当前用户对应的客户端发送消息
//
//	func (u *User) SendMsg(msg string) {
//		if err := u.conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
//			log.Printf("SendMsg error to %s: %v\n", u.Name, err)
//			u.Offline()
//		}
//	}
func (u *User) SendJSON(m Message) {
	data, _ := json.Marshal(m)
	u.C <- string(data)
}

// 处理用户消息
//
//	func (u *User) DoMessage(msg string) {
//		if msg == "who" {
//			u.server.mapLock.Lock()
//			for _, user := range u.server.OnlineMap {
//				onlineMsg := "[" + user.Name + "]" + ":online..."
//				u.C <- onlineMsg
//			}
//			u.server.mapLock.Unlock()
//		} else if len(msg) > 7 && msg[:7] == "rename|" {
//			newName := strings.Split(msg, "|")[1]
//			u.server.mapLock.Lock()
//			if _, ok := u.server.OnlineMap[newName]; ok {
//				u.SendMsg("the name already exists")
//			} else {
//				delete(u.server.OnlineMap, u.Name)
//				u.server.OnlineMap[newName] = u
//				u.Name = newName
//				u.SendMsg("you have changed your name to:" + u.Name)
//			}
//			u.server.mapLock.Unlock()
//		} else if len(msg) > 4 && msg[:3] == "to|" {
//			parts := strings.SplitN(msg, "|", 3)
//			if len(parts) < 3 {
//				u.SendMsg("please use correct format: to|username|message content")
//				return
//			}
//			u.server.mapLock.Lock()
//			defer u.server.mapLock.Unlock()
//			remoteName := parts[1]
//			content := parts[2]
//			if remoteUser, ok := u.server.OnlineMap[remoteName]; ok {
//				remoteUser.SendMsg(u.Name + " to you:" + content)
//			} else {
//				u.SendMsg("the user does not exist")
//			}
//		} else {
//			u.server.Broadcast(u, msg)
//		}
//	}
func (u *User) DoMessage(msg string) {
	var m Message
	if err := json.Unmarshal([]byte(msg), &m); err != nil {
		log.Printf("DoMessage unmarshal error from %s: %v\n", u.Name, err)
		return
	}
	switch m.Type {
	case "chat": // 修改：从 "public" 改为 "chat"
		publicMsg := Message{Type: "chat", From: u.Name, Content: m.Content} // 修改：广播类型也改为 "chat"
		u.server.Broadcast(publicMsg)

	case "private":
		u.server.mapLock.RLock()
		targetUser, ok := u.server.OnlineMap[m.To]
		u.server.mapLock.RUnlock()
		if ok {
			privateMsg := Message{Type: "private", From: u.Name, To: m.To, Content: m.Content}
			targetUser.SendJSON(privateMsg)
		} else {
			u.SendJSON(Message{Type: "system", Content: "user does not exist"})
		}

	case "rename":
		newName := m.NewName // 前端发送的改名消息是 {type: 'rename', newName: 'xxx'}，后端这里要适配
		if newName == "" {   // 加个简单判断
			u.SendJSON(Message{Type: "system", Content: "new name cannot be empty"})
			return
		}
		u.server.mapLock.Lock()
		if _, ok := u.server.OnlineMap[newName]; ok {
			u.SendJSON(Message{Type: "system", Content: "the name already exists"})
		} else {
			delete(u.server.OnlineMap, u.Name)
			u.server.OnlineMap[newName] = u
			oldName := u.Name
			u.Name = newName
			// 广播改名成功消息给前端，让它更新用户名显示
			u.SendJSON(Message{Type: "rename", NewName: u.Name})
			// 同时广播一条系统消息
			sysMsg := Message{Type: "system", Content: fmt.Sprintf("%s has renamed to %s", oldName, u.Name)}
			u.server.Broadcast(sysMsg)
		}
		u.server.mapLock.Unlock()

	case "who":
		u.server.mapLock.Lock()
		var userList []string
		for _, user := range u.server.OnlineMap {
			userList = append(userList, user.Name)
		}
		u.server.mapLock.Unlock()
		// 返回用户列表，符合前端期望的 {type: 'user_list', users: [...]}
		u.SendJSON(Message{Type: "user_list", Content: "", Users: userList})
	}
}

// 监听当前User channel的方法，一旦有消息，就直接发送给客户端
func (u *User) ListenMessage() {
	for msg := range u.C {
		u.conn.WriteMessage(websocket.TextMessage, []byte(msg))
	}
}
