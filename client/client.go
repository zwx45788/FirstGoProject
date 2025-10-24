package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
)

type Client struct {
	ServerAddr string // 改为使用一个地址字符串
	Name       string
	Conn       *websocket.Conn // 关键改动：从 net.Conn 改为 websocket.Conn
	flag       int             // 当前客户端模式
}

func NewClient(serverAddr string) *Client {
	// 解析WebSocket地址
	u, err := url.Parse(serverAddr)
	if err != nil {
		log.Println("地址解析失败:", err)
		return nil
	}

	// 连接WebSocket服务器
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		fmt.Println("连接服务器失败:", err)
		return nil
	}

	client := &Client{
		ServerAddr: serverAddr,
		Conn:       conn,
		flag:       999,
	}
	return client
}

func (c *Client) menu() bool {
	var flag int

	fmt.Println("1.公共聊天模式")
	fmt.Println("2.私聊模式")
	fmt.Println("3.更新用户名")
	fmt.Println("0.退出")

	fmt.Scanln(&flag)

	if flag >= 0 && flag <= 3 {
		c.flag = flag
		return true
	} else {
		fmt.Println("请输入有效数字")
		return false
	}
}

func (c *Client) updateName() bool {
	fmt.Println("请输入新用户名:")
	fmt.Scanln(&c.Name)

	// 关键改动：不再需要 "\r\n"
	sendMsg := "rename|" + c.Name
	err := c.Conn.WriteMessage(websocket.TextMessage, []byte(sendMsg))
	if err != nil {
		fmt.Println("发送消息失败:", err)
		return false
	}
	return true
}

func (c *Client) PublicChat() {
	var chatMsg string
	fmt.Println("请输入聊天内容, 输入 exit 退出当前模式")
	fmt.Scanln(&chatMsg)

	for chatMsg != "exit" {
		if len(chatMsg) != 0 {
			// 关键改动：直接发送文本内容
			err := c.Conn.WriteMessage(websocket.TextMessage, []byte(chatMsg))
			if err != nil {
				fmt.Println("发送消息失败:", err)
				return
			}
		}
		chatMsg = ""
		fmt.Println("请输入聊天内容, 输入 exit 退出当前模式")
		fmt.Scanln(&chatMsg)
	}
}

// 这个函数需要重写，因为WebSocket不能像TCP那样用io.Copy
func (c *Client) DealResponse() {
	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			fmt.Println("\n与服务器连接断开")
			os.Exit(1)
			return
		}
		// 为了不让菜单和接收消息混在一起，我们加个换行
		fmt.Print("\n" + string(msg) + "\n> ")
	}
}

func (c *Client) SelectUsers() {
	// 关键改动：发送 "who" 命令
	sendMsg := "who"
	err := c.Conn.WriteMessage(websocket.TextMessage, []byte(sendMsg))
	if err != nil {
		fmt.Println("发送消息失败:", err)
		return
	}
}

func (c *Client) PrivateChat() {
	c.SelectUsers()
	var remoteName string
	fmt.Println("请输入私聊对象, 输入 exit 退出")
	fmt.Scanln(&remoteName)

	for remoteName != "exit" {
		fmt.Println("请输入聊天内容, 输入 exit 退出私聊")
		var chatMsg string
		fmt.Scanln(&chatMsg)

		for chatMsg != "exit" {
			if len(chatMsg) != 0 {
				// 关键改动：发送 "to|用户|消息" 格式
				sendMsg := "to|" + remoteName + "|" + chatMsg
				err := c.Conn.WriteMessage(websocket.TextMessage, []byte(sendMsg))
				if err != nil {
					fmt.Println("发送消息失败:", err)
					break
				}
			}
			chatMsg = ""
			fmt.Println("请输入聊天内容, 输入 exit 退出私聊")
			fmt.Scanln(&chatMsg)
		}
		c.SelectUsers()
		fmt.Println("请输入私聊对象, 输入 exit 退出")
		fmt.Scanln(&remoteName)
	}
}

func (c *Client) Run() {
	for c.menu() {
		switch c.flag {
		case 1:
			c.PublicChat()
		case 2:
			c.PrivateChat()
		case 3:
			c.updateName()
		}
	}
}

var serverIp string
var serverPort int

// 命令行解析
func init() {
	flag.StringVar(&serverIp, "ip", "127.0.0.1", "设置服务器IP地址(默认值为127.0.0.1)")
	flag.IntVar(&serverPort, "port", 8081, "设置服务器端口(默认值为8081)") // 关键改动：默认端口改为8081
}

func main() {
	flag.Parse()

	// 关键改动：拼接成WebSocket地址
	serverAddr := fmt.Sprintf("ws://%s:%d/ws", serverIp, serverPort)
	client := NewClient(serverAddr)
	if client == nil {
		fmt.Println("创建客户端失败")
		return
	}
	defer client.Conn.Close()

	// 启动一个goroutine来处理服务器响应
	go client.DealResponse()

	fmt.Println("成功连接到服务器")
	client.Run()
}
