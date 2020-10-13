package GC

import (
	"log"
	cmp "marvin/GraphEng/Compression"
)

/**
Protocoll:
Client registers SyncVar -> 		Payload:
[SYNCVAR_REGISTRY 				| SyncVar.Type() 		| (n){name of SyncVar}]													len() = 2+n

-> Server sends Confirmation -> 	Payload:
[SYNCVAR_REGISTRY_CONFIRMATION 	| length of the name 	| (n){name} 				| (2){id}]									len() = 4+n

Server registers SyncVar -> 		Payload:
[SYNCVAR_REGISTRY 				| SyncVar.Type() 		| length of the name 		| (n){name} 		| (2){id}]				len() = 5+n

SyncVars Update by Client/Server -> Payload:
[SYNCVAR_UPDATE					| (n1){ (2)(id)			| (2)(SyncVar Length) 		| (n2)(SyncVar Data) }]						len() = 1+n*(4+n2)
//1 Int64 --> 13 byte

SyncVar Delete -> 					Payload:
[SYNCVAR_DELETION				| (2){id} 				| (n){name}]															len() = 2+n
**/

func GetClientManager(c *Client) (cm *ClientManager) {
	cm = &ClientManager{Client:c}
	cm.SyncvarsByID = 	make(map[int]SyncVar)
	cm.SyncvarsByName = make(map[string]SyncVar)
	cm.NameToID	=		make(map[string]int)
	c.InputHandler = 	cm.receive
	return
}
type ClientManager struct {
	Client			*Client
	SyncvarsByID	map[int]SyncVar
	SyncvarsByName	map[string]SyncVar
	NameToID		map[string]int
	
	InputHandler  func(mt int, msg []byte, err error, c *Client) (alive bool)
}
func (m *ClientManager) RegisterSyncVar(sv SyncVar, name string) {
	if m.SyncvarsByID == nil {
		m.SyncvarsByID = make(map[int]SyncVar)
	}
	if m.SyncvarsByName == nil {
		m.SyncvarsByName = make(map[string]SyncVar)
	}
	m.SyncvarsByName[name] = sv
	m.Client.Send(append([]byte{SYNCVAR_REGISTRY, sv.Type()}, []byte(name)...))
}
func (m *ClientManager) UpdateSyncVars() {
	var_data := []byte{SYNCVAR_UPDATE}
	for id,sv := range(m.SyncvarsByID) {
		if sv.isDirty() {
			syncDat := sv.getData()
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			payload := append(cmp.Int16ToBytes(int16(id)), data...)
			var_data = append(var_data, payload...)
		}
	}
	m.Client.Send(var_data)
}
func (m *ClientManager) DeleteSyncVar(name string) {
	id := m.NameToID[name]
	m.deleteSyncVarLocal(name, id)
	m.deleteSyncVarRemote(name, id)
}
func (m *ClientManager) deleteSyncVarLocal(name string, id int) {
	delete(m.SyncvarsByID, id)
	delete(m.SyncvarsByName, name)
	delete(m.NameToID, name)
}
func (m *ClientManager) deleteSyncVarRemote(name string, id int) {
	data := append(cmp.Int16ToBytes(int16(id)), []byte(name)...)
	m.Client.Send(append([]byte{SYNCVAR_DELETION}, data...))
}
func (m *ClientManager) receive(mt int, input []byte, err error, c *Client) bool {
	if err != nil {
		return false
	}
	if input[0] == SYNCVAR_REGISTRY_CONFIRMATION {
		m.processRegisterVarConfirm(input[1:])
	}else if input[0] == SYNCVAR_REGISTRY {
		l := input[2]
		t := input[1]
		name := string(input[3:3+l])
		id := int(cmp.BytesToInt16(input[3+l:]))
		log.Printf("Client: Creating SyncVar with ID=%v, name='%s' , initiated by server=%s", id, name, m.Client.RemoteAddr().String())
		m.SyncvarsByName[name] = GetSyncVarOfType(t)
		m.SyncvarsByID[id] = m.SyncvarsByName[name]
		m.NameToID[name] = id
	}else if input[0] == SYNCVAR_UPDATE {
		m.onSyncVarUpdateC(input[1:])
	}else if input[0] == SYNCVAR_DELETION {
		id := int(cmp.BytesToInt16(input[1:3]))
		name := string(input[3:])
		m.deleteSyncVarLocal(name, id)
		log.Printf("Client: Deleting SyncVar with ID=%v, name='%s' , initiated by server=%s", id, name, m.Client.RemoteAddr().String())
	}
	if m.InputHandler != nil {
		return m.InputHandler(mt, input, err, c)
	}
	return true
}
func (m *ClientManager) processRegisterVarConfirm(data []byte) {
	l := data[0]; data = data[1:]
	name := string(data[:l])
	id := int(cmp.BytesToInt16(data[l:]))
	m.SyncvarsByID[id] = m.SyncvarsByName[name]
	m.NameToID[name] = id
	log.Printf("Client: Creating SyncVar with ID=%v, name='%s' , confirmed by server=%s", id, name, m.Client.RemoteAddr().String())
}
func (m *ClientManager) onSyncVarUpdateC(data []byte) {
	//log.Printf("Updating SyncVars with data: ", data)
	for true {
		id := int(cmp.BytesToInt16(data[:2]))
		l := cmp.BytesToInt16(data[2:4])
		dat := data[4:4+l]
		log.Printf("Client: Updating SyncVar with ID=%v: len(dat)=%v, dat=%v", id, l, dat)
		data = data[4+l:]
		m.SyncvarsByID[id].setData(dat)
		if len(data) <= 0 {
			break
		}
	}
}
