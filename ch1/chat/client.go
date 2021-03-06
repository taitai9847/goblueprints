package main

import (
	"time"

	"github.com/gorilla/websocket"
)

type client struct {
	// クライアントのためのwebsocket
	socket *websocket.Conn
	// sendはメッセージが送られるチャネルです
	send chan *message
	// roomはこのクライアントが参加しているroom
	room *room
	// userDataはユーザーに関する情報
	userData map[string]interface{}
}

func (c *client) read() {
	defer c.socket.Close()
	for {
		var msg *message
		err := c.socket.ReadJSON(&msg)
		if err != nil {
			return
		}
		msg.When = time.Now()
		msg.Name = c.userData["name"].(string)
		if avatarURL, ok := c.userData["avatar_url"]; ok {
			msg.AvatarURL = avatarURL.(string)
		}
		c.room.forward <- msg
	}
}

func (c *client) write() {
	for msg := range c.send {
		if err := c.socket.WriteJSON(msg); err != nil {
			break
		}
	}
	c.socket.Close()
}
