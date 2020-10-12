package main

import (
	"fmt"
	"marvin/GameConn/GC"
	"time"

	ws "github.com/gorilla/websocket"
)

func main() {
	client := GC.GetNewClient("FetterClient")
	clientmanager := GC.ClientManager{Client: client}

	client.InputHandler = func(mt int, msg []byte, err error, c *GC.Client) (alive bool) {
		clientmanager.Receive(msg)
		return true
	}

	server := GC.GetNewServer()
	servermanager := GC.ServerManager{Server: server}

	server.InputHandler = ServerInput
	server.OnNewConn = ServerNewConn
	server.OnCloseConn = ServerCloseConn

	server.Run("localhost:8080")
	err1 := client.MakeConn("localhost:8080")
	if err1 != nil {
		panic(err1)
	}

	servervariable := GC.CeateSyncInt64(0)
	servermanager.RegisterVar(servervariable)

	clientvariable := GC.CeateSyncInt64(0)
	clientmanager.RegisterVar(clientvariable)

	servervariable.SetInt(35)
	servermanager.Update()

	time.Sleep(time.Second * 3)
	fmt.Printf("%v \n", clientvariable.GetInt())
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
