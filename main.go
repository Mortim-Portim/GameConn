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
	
	
	syncVar1 := GC.CeateSyncInt64(35)
	syncVar2 := GC.CeateSyncFloat64(35.35)
	syncVar3 := GC.CeateSyncString("fünf und dreißig")
	
	servermanager.RegisterSyncVar(syncVar1, "Fette Test Int64 Variable", servermanager.AllClients...)
	server.WaitForAllConfirmations()
	
	servermanager.RegisterSyncVar(syncVar2, "Fette Test Float64 Variable", servermanager.AllClients...)
	server.WaitForAllConfirmations()
	
	servermanager.RegisterSyncVar(syncVar3, "Fette Test String Variable", servermanager.AllClients...)
	server.WaitForAllConfirmations()
	
	servermanager.UpdateSyncVars()
	server.WaitForAllConfirmations()
	
	fmt.Println(len(servermanager.GetHandler(0).SyncvarsByName))
	fmt.Println(len(clientmanager.SyncvarsByName))
	
	clientmanager.DeleteSyncVar("Fette Test Int64 Variable")
	client.WaitForConfirmation()
	
	fmt.Println(len(servermanager.GetHandler(0).SyncvarsByName))
	fmt.Println(len(clientmanager.SyncvarsByName))

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
	//fmt.Println("Client Disconnected: ", c.RemoteAddr().String())

}

func ClientInput(mt int, msg []byte, err error, c *GC.Client) bool {
	//fmt.Printf("Client received: %s\n", string(msg))

	if err != nil {
		return false
	}
	return true
}
