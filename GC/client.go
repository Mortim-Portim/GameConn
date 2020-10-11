package GC

import (
	"net/url"
	"os"
	"os/signal"
	"time"

	ws "github.com/gorilla/websocket"
)


func GetNewClient(InputHandler func(mt int, msg []byte, err error, c *Client) (alive bool), name string) (cl *Client) {
	cl = &Client{name:name, InputHandler:InputHandler}
	return
}

type Client struct {
	ws.Conn
	name string
	done, waiting chan struct{}
	interrupt chan os.Signal
	InputHandler func(mt int, msg []byte, err error, c *Client) (alive bool)
	sendMessage []byte
}
func (cl *Client) Send(bs []byte) {
	cl.sendMessage = bs
	close(cl.waiting)
}

//addr := "localhost:8080"
func (c *Client) MakeConn(addr string) error {
	//If the system exits c.interrupt fires
	c.interrupt = make(chan os.Signal, 1)
	signal.Notify(c.interrupt, os.Interrupt)
	
	//Create an URL to dial
	u := url.URL{Scheme: "ws", Host: addr, Path: "/echo"}
	
	//dial URL and set client connection
	c_tmp, _, err := ws.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	c.Conn = *c_tmp
	
	//receive input in a separate thread
	c.done = make(chan struct{})
	go func() {
		defer close(c.done)
		for {
			mt, msg, err := c.ReadMessage()
			if !c.InputHandler(mt, msg, err, c) {
				return
			}
		}
	}()
	
	//Send an initial message
	err = c.WriteMessage(ws.TextMessage, []byte(NEWCONN))
	if err != nil {
		return err
	}
	
	//wait for input an send it on a separate thread
	c.waiting = make(chan struct{})
	go func() {
		for {
			select {
				case <-c.done:
					c.CloseConn()
					return
				case <-c.waiting:
					if c.sendMessage != nil {
						err := c.WriteMessage(ws.BinaryMessage, c.sendMessage)
						if err != nil {
							return
						}
						c.waiting = make(chan struct{})
					}
				case <-c.interrupt:
					c.CloseConn()
					return
			}
		}
	}()
	return nil
}

func (cl *Client) CloseConn() error {
	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	err := cl.WriteMessage(ws.CloseMessage, []byte(CLOSECONN))
	if err != nil {
		return err
	}
	select {
		case <-cl.done:
		case <-time.After(time.Second):
	}
	return nil
}