package GC

import (
	"net/url"
	"os"
	"os/signal"
	"time"
	ws "github.com/gorilla/websocket"
)

func GetNewClient() (cl *Client) {
	cl = &Client{confirmed:make(chan bool)}
	return
}
type Client struct {
	ws.Conn
	done, waiting chan struct{}
	interrupt     chan os.Signal
	InputHandler  func(mt int, msg []byte, err error, c *Client) (alive bool)
	sendMessage   []byte
	confirmed	  chan bool
	pendingConfirms int
}
func (c *Client) Send(bs []byte) {
	time.Sleep(ARTIFICIAL_CLIENT_PING)
	c.pendingConfirms ++
	c.sendMessage = bs
	close(c.waiting)
}
func (c *Client) WaitForConfirmation() {
	for c.pendingConfirms > 0 {
		<-c.confirmed
		c.pendingConfirms --
	}
	return
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
	
//	c.Conn.SetPingHandler(func(appData string) error {
//		log.Println("Client Ping")
//		err := c.WriteControl(ws.PongMessage, []byte(appData), time.Now().Add(time.Second))
//		if err == ws.ErrCloseSent {
//			return nil
//		} else if e, ok := err.(net.Error); ok && e.Temporary() {
//			return nil
//		}
//		return err
//	})
	c.Conn.SetPongHandler(func(appData string) error {
		if c.confirmed != nil {
			c.confirmed <- true
		}
		
		return nil
	})

	//receive input in a separate thread
	c.done = make(chan struct{})
	go func() {
		defer close(c.done)
		for {
			if c != nil && c.InputHandler != nil {
				mt, msg, err := c.ReadMessage()
				if err != nil {
					return
				}
				if !c.InputHandler(mt, msg, err, c) {
					return
				}
				err2 := c.WriteMessage(ws.PongMessage, []byte{})
				if err2 != nil {
					return
				}
			}
		}
	}()

	//Send an initial message
	err = c.WriteMessage(ws.TextMessage, []byte{NEWCONNECTION})
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
					c.sendMessage = nil
				}
			case <-c.interrupt:
				c.CloseConn()
				return
			}
		}
	}()
	time.Sleep(time.Millisecond)
	return nil
}
//Should only be called with a delay
func (c *Client) CloseConn() error {
	err := c.WriteMessage(ws.BinaryMessage, []byte{CLOSECONNECTION})
	if err != nil {
		return err
	}
	time.Sleep(time.Second)
	return nil
}