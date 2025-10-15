package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type MessageHandler interface {
	handleChatMessage(*Client, *IncomingMessage)
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan *Message
	sessionID string
	username  string
	geohash   string
	radius    int
	ctx       context.Context
	cancel    context.CancelFunc
	handler   MessageHandler  // Add this line

}

func NewClient(hub *Hub, conn *websocket.Conn, sessionID, username, geohash string, radius int, handler MessageHandler) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan *Message, 256),
		sessionID: sessionID,
		username:  username,
		geohash:   geohash,
		radius:    radius,
		ctx:       ctx,
		cancel:    cancel,
		handler:   handler,  // Add this line
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		c.cancel()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// for {
	// 	_, message, err := c.conn.ReadMessage()
	// 	if err != nil {
	// 		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
	// 			log.Printf("error: %v", err)
	// 		}
	// 		break
	// 	}

	// 	var msg IncomingMessage
	// 	if err := json.Unmarshal(message, &msg); err != nil {
	// 		log.Printf("error unmarshaling message: %v", err)
	// 		continue
	// 	}

	// 	c.handleIncomingMessage(&msg)
	// }


	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
	
		var msg IncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("error unmarshaling message: %v", err)
			continue
		}

		fmt.Println("msg type", msg.Type, msg.Content)
	
		// Handle message based on type
		switch msg.Type {
		case MessageTypeChat:
			if c.handler != nil {
				c.handler.handleChatMessage(c, &msg)
			}
		case MessageTypePing:
			pong := &Message{
				Type:      MessageTypePong,
				Timestamp: time.Now().Unix(),
			}
			c.send <- pong
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("error marshaling message: %v", err)
				w.Close()
				continue
			}

			w.Write(data)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				msg := <-c.send
				data, err := json.Marshal(msg)
				if err != nil {
					continue
				}
				w.Write([]byte("\n"))
				w.Write(data)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// func (c *Client) handleIncomingMessage(msg *IncomingMessage) {
// 	switch msg.Type {
// 	case MessageTypeChat:
// 		// This will be handled by the handler
// 		// The handler will validate and broadcast
// 	case MessageTypePing:
// 		// Respond with pong
// 		pong := &Message{
// 			Type:      MessageTypePong,
// 			Timestamp: time.Now().Unix(),
// 		}
// 		c.send <- pong
// 	}
// }

func (c *Client) shouldReceiveMessage(msg *Message) bool {
	// Check if message is in client's geohash vicinity
	if c.geohash == "" {
		return false
	}

	// Simple proximity check - in production, use proper distance calculation
	return c.geohash[:4] == msg.Geohash[:4]
}

func (c *Client) UpdateLocation(geohash string, radius int) {
	c.geohash = geohash
	c.radius = radius
}

func (c *Client) UpdateUsername(username string) {
	c.username = username
}

func (c *Client) SendError(errMsg string, code string) {
	msg := &Message{
		Type:      MessageTypeError,
		Content:   errMsg,
		ErrorCode: code,
		Timestamp: time.Now().Unix(),
	}
	select {
	case c.send <- msg:
	default:
	}
}