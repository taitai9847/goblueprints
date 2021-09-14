package main

import (
	"github.com/gorilla/websocket"
)

type client struct {
	// クライアントのためのwebsocket
	socket *websocket.Conn
	// sendはメッセージが送られるチャネルです
	send chan []byte
	// roomはこのクライアントが参加しているroom
	room *room
}

func (c *client) read() {
	for {
		if _, msg, err := c.socket.ReadMessage(); err == nil {
			c.room.forward <- msg
		} else {
			break
		}
	}
}

func (c *client) write() {
	for msg := range c.send {
		if err := c.socket.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
	c.socket.Close()
}
