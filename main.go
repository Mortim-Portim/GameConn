package main

import (
	"fmt"
	"marvin/GameConn/GC"
	"bytes"
	"encoding/gob"

	ws "github.com/gorilla/websocket"
)


/**
Point is used for testing gob.Encode to encode a struct
a struct of a float64 is encoded to 38 bytes, which is 30 bytes more than needed by Float64Sync
**/
var PointEncoder *gob.Encoder
var PointDecoder *gob.Decoder
var codeBuffer bytes.Buffer
//Registers Encoder/Decoder to a buffer
func InitPoints() {
	PointEncoder = gob.NewEncoder(&codeBuffer)
	PointDecoder = gob.NewDecoder(&codeBuffer)
}
type Pnt struct {
	X float64
}
type Point struct {
	pnt *Pnt
	isdirty bool
}
func (p *Point) IsDirty() bool {
	return p.isdirty
}
//Encodes the data of pnt
func (p *Point) GetData() []byte {
	defer codeBuffer.Reset()
	err := PointEncoder.Encode(p.pnt)
	if err != nil {panic(err)}
	fmt.Println("Encoded Bytes: ", codeBuffer.Bytes())
	return codeBuffer.Bytes()
}
//Decodes the data of pnt
func (p *Point) SetData(bs []byte) {
	codeBuffer.Reset()
	codeBuffer.Write(bs)
	fmt.Println("Decoded Bytes: ", codeBuffer.Bytes())
	err := PointDecoder.Decode(p.pnt)
	if err != nil {panic(err)}
}
func (p *Point) Type() byte {
	return byte(3)
}
func GetPoint() GC.SyncVar {
	return &Point{pnt:&Pnt{}}
}

func main() {
	//Initializes Standard types such as float64, int64, string
	GC.InitSyncVarStandardTypes()
	//Registers Points as a SyncVar
	InitPoints()
	GC.RegisterSyncVar(3, GetPoint)
	
	//Creates a client and clientmanager
	client := GC.GetNewClient()
	clientmanager := GC.GetClientManager(client)
	clientmanager.InputHandler = ClientInput
	
	//Creates a server and servermanager
	server := GC.GetNewServer()
	servermanager := GC.GetServerManager(server)
	servermanager.InputHandler = ServerInput
	servermanager.OnNewConn = ServerNewConn
	servermanager.OnCloseConn = ServerCloseConn

	//Runs the server
	server.Run("localhost:8080")
	//Connects the client to the server
	err1 := client.MakeConn("localhost:8080")
	if err1 != nil {
		panic(err1)
	}
	
	//Creates three syncvars
	syncVar1 := &Point{&Pnt{25.3}, true}
	syncVar2 := GC.CreateSyncFloat64(35.35)
	syncVar3 := GC.CreateSyncString("fünf und dreißig")
	
	//Registers a syncvar and waits for the managers to finish communication
	servermanager.RegisterSyncVar(syncVar1, "Fette Test Point Variable", servermanager.AllClients...)
	server.WaitForAllConfirmations()
	//Registers a syncvar and waits for the managers to finish communication
	servermanager.RegisterSyncVar(syncVar2, "Fette Test Float64 Variable", servermanager.AllClients...)
	server.WaitForAllConfirmations()
	//Registers a syncvar and waits for the managers to finish communication
	servermanager.RegisterSyncVar(syncVar3, "Fette Test String Variable", servermanager.AllClients...)
	server.WaitForAllConfirmations()
	
	//Updates all syncvars that changed on the server side and waits for the managers to finish communication
	servermanager.UpdateSyncVars()
	server.WaitForAllConfirmations()
	
	//Prints the Point syncvar of the server and client
	fmt.Println(servermanager.GetHandler(0).SyncvarsByName["Fette Test Point Variable"])
	fmt.Println(clientmanager.SyncvarsByName["Fette Test Point Variable"])
	
	//Deletes the point syncvar and waits for the managers to finish communication
	clientmanager.DeleteSyncVar("Fette Test Point Variable")
	client.WaitForConfirmation()
	
	//shows that both manager have one syncvar less
	fmt.Println(servermanager.GetHandler(0).SyncvarsByName)
	fmt.Println(clientmanager.SyncvarsByName)

	client.CloseConn()
	fmt.Println("finished")
}

func ServerInput(c *ws.Conn, mt int, msg []byte, err error, s *GC.Server) {
	//fmt.Printf("Client %s send: %s\n", c.LocalAddr().String(), msg)

}
func ServerNewConn(c *ws.Conn, mt int, msg []byte, err error, s *GC.Server) {
	//fmt.Println("New Client Connected: ", c.LocalAddr().String())

}
func ServerCloseConn(c *ws.Conn, mt int, msg []byte, err error, s *GC.Server) {
	fmt.Println("Client Disconnected: ", c.RemoteAddr().String())

}

func ClientInput(mt int, msg []byte, err error, c *GC.Client) bool {
	//fmt.Printf("Client received: %s\n", msg)

	if err != nil {
		return false
	}
	return true
}
