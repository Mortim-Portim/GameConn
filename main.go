package main

import (
	"fmt"
	"marvin/GameConn/GC"
	"time"

	ws "github.com/gorilla/websocket"
)

func main() {
	client := GC.GetNewClient("FetterClient")
	clientmanager := GC.GetClientManager(client)
	clientmanager.InputHandler = ClientInput
	
	server := GC.GetNewServer()
	servermanager := GC.GetServerManager(server)
	servermanager.InputHandler = ServerInput
	servermanager.OnNewConn = ServerNewConn
	servermanager.OnCloseConn = ServerCloseConn

	server.Run("localhost:8080")
	//time.Sleep(time.Second)
	err1 := client.MakeConn("localhost:8080")
	if err1 != nil {
		panic(err1)
	}
	
	
	syncVar := GC.CeateSyncInt64(0)
	syncVar.SetInt(35)
	servermanager.RegisterSyncVar(syncVar, "Fette Test Variable", servermanager.AllClients...)
	
	//client.CloseConn()
	
	time.Sleep(time.Second)
	servermanager.UpdateSyncVars()
	
	time.Sleep(time.Second)
	fmt.Println(servermanager.GetHandler(0).SyncvarsByName["Fette Test Variable"].(*GC.SyncInt64).GetInt())
	fmt.Println(clientmanager.SyncvarsByName["Fette Test Variable"].(*GC.SyncInt64).GetInt())
	
	time.Sleep(time.Second)
	clientmanager.DeleteSyncVar("Fette Test Variable")
	
	time.Sleep(time.Second)
	fmt.Println(servermanager.GetHandler(0).SyncvarsByName["Fette Test Variable"])
	fmt.Println(clientmanager.SyncvarsByName["Fette Test Variable"])

	time.Sleep(time.Second * 3)
	fmt.Println("finished")
}

func ServerInput(c *ws.Conn, mt int, msg []byte, err error, s *GC.Server) {
	//fmt.Printf("Client %s send: %s\n", c.LocalAddr().String(), string(msg))

}
func ServerNewConn(c *ws.Conn, mt int, msg []byte, err error, s *GC.Server) {
	//fmt.Println("New Client Connected: ", c.LocalAddr().String())

}
func ServerCloseConn(c *ws.Conn, mt int, msg []byte, err error, s *GC.Server) {
	fmt.Println("Client Disconnected: ", c.RemoteAddr().String())

}

func ClientInput(mt int, msg []byte, err error, c *GC.Client) bool {
	//fmt.Printf("Client received: %s\n", string(msg))

	if err != nil {
		return false
	}
	return true
}
