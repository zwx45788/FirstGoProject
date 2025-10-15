package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

type Client struct {
	ServerIp   string
	ServerPort int
	Name       string
	Conn       net.Conn
	flag       int //当前客户端模式
}

func NewClient(serverIp string, serverPort int) *Client {
	Client := &Client{
		ServerIp:   serverIp,
		ServerPort: serverPort,
		flag:       999,
	}
	//连接server
	conn, err := net.Dial("tcp", net.JoinHostPort(serverIp, fmt.Sprintf("%d", serverPort)))
	if err != nil {
		fmt.Println("连接服务器失败:", err)
		return nil
	}
	Client.Conn = conn
	return Client
}

func (c *Client) menu() bool {
	var flag int

	fmt.Println("1.public chat mode")
	fmt.Println("2.private chat mode")
	fmt.Println("3.update username")
	fmt.Println("0.exit")

	fmt.Scanln(&flag)

	if flag >= 0 && flag <= 3 {
		c.flag = flag
		return true
	} else {
		fmt.Println("please input valid number")
		return false
	}
}
func (c *Client) updateName() bool {
	fmt.Println("please input your name:")
	fmt.Scanln(&c.Name)

	sendMsg := "rename|" + c.Name + "\r\n"
	_, err := c.Conn.Write([]byte(sendMsg))
	if err != nil {
		fmt.Println("conn.Write err:", err)
		return false
	}
	return true
}
func (c *Client) PublicChat() {
	var chatMsg string
	fmt.Println("please input chat content, exit to quit")
	fmt.Scanln(&chatMsg)

	for chatMsg != "exit" {
		if len(chatMsg) != 0 {
			sendMsg := chatMsg + "\r\n"
			_, err := c.Conn.Write([]byte(sendMsg))
			if err != nil {
				fmt.Println("conn.Write err:", err)
				return
			}
		}
		chatMsg = ""
		fmt.Println("please input chat content, exit to quit")
		fmt.Scanln(&chatMsg)
	}
}
func (c *Client) DealResponse() {
	io.Copy(os.Stdout, c.Conn)
}
func (c *Client) SelectUsers() {
	sendMsg := "who\r\n"
	_, err := c.Conn.Write([]byte(sendMsg))
	if err != nil {
		fmt.Println("conn.Write err:", err)
		return
	}
}
func (c *Client) PrivateChat() {
	c.SelectUsers()
	var remoteName string
	fmt.Println("please input the username you want to chat with, exit to quit")
	fmt.Scanln(&remoteName)
	for remoteName != "exit" {
		fmt.Println("please input chat content, exit to quit")
		var chatMsg string
		fmt.Scanln(&chatMsg)
		for chatMsg != "exit" {
			if len(chatMsg) != 0 {
				sendMsg := "to|" + remoteName + "|" + chatMsg + "\r\n"
				_, err := c.Conn.Write([]byte(sendMsg))
				if err != nil {
					fmt.Println("conn.Write err:", err)
					break
				}
			}
			chatMsg = ""
			fmt.Println("please input chat content, exit to quit")
			fmt.Scanln(&chatMsg)
		}
		c.SelectUsers()
		fmt.Println("please input the username you want to chat with, exit to quit")
		fmt.Scanln(&remoteName)
	}
}
func (c *Client) Run() {
	for c.menu() {
		switch c.flag {
		case 1:
			//fmt.Println("public chat mode...")
			c.PublicChat()
		case 2:
			//fmt.Println("private chat mode...")
			c.PrivateChat()
		case 3:
			c.updateName()
			//fmt.Println("update username...")
		}
	}
}

var serverIp string
var serverPort int

// 命令行解析
func init() {
	flag.StringVar(&serverIp, "ip", "127.0.0.1", "设置服务器IP地址(默认值为127.0.0.1)")
	flag.IntVar(&serverPort, "port", 8080, "设置服务器端口(默认值为8080)")
}
func main() {

	flag.Parse()
	Client := NewClient(serverIp, serverPort)
	if Client == nil {
		fmt.Println("fail to create client")
		return
	}
	defer Client.Conn.Close()

	go Client.DealResponse()
	fmt.Println("success to connect server")
	Client.Run()
}
