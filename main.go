package main 

import (
	"fmt"
	"time"
	"marvin/GameConn/GC"
	ws "github.com/gorilla/websocket"
)

func main() {
	client := GC.GetNewClient(ClientInput, "FetterClient")
	
	server := GC.GetNewServer()
	server.InputHandler = ServerInput
	server.OnNewConn = ServerNewConn
	server.OnCloseConn = ServerCloseConn
	
	server.Run("localhost:8080")
	err1 := client.MakeConn("localhost:8080")
	if err1 != nil {
		panic(err1)
	}
	
	server.Send([]byte("Hallo ich bin fett"), 0)
	client.Send([]byte("Ach echt, ich auch"))
	
	time.Sleep(time.Second*3)
	fmt.Println("finished")
}

func ServerInput(c *ws.Conn, mt int, msg []byte, err error, s *GC.Server) {
	fmt.Printf("Client %s send: %s\n", c.LocalAddr().String(), string(msg))
	
}
func ServerNewConn(c *ws.Conn, mt int, msg []byte, err error, s *GC.Server) {
	fmt.Println("New Client Connected: ", c.LocalAddr().String())
	
}
func ServerCloseConn(c *ws.Conn, mt int, msg []byte, err error, s *GC.Server) {
	fmt.Println("Client Disconnected: ", c.LocalAddr().String())
	
}

func ClientInput(mt int, msg []byte, err error, c *GC.Client) bool {
	fmt.Printf("Client received: %s\n", string(msg))
	
	if err != nil {
		return false
	}
	return true
}