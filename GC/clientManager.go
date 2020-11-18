package GC

import (
	cmp "github.com/mortim-portim/GraphEng/Compression"
)

/*
Protocoll:
Client registers SyncVar -> 		Payload:
[SYNCVAR_REGISTRY 				| SyncVar.Type() 		| (1){AccessID}]														len() = 3

-> Server sends Confirmation -> 	Payload:
[SYNCVAR_REGISTRY_CONFIRMATION 	| (1)AccessID] 			| (2){id}]																		len() = 4

Server registers SyncVar -> 		Payload:
[SYNCVAR_REGISTRY 				| SyncVar.Type() 		| (1)AccessID 				| (2){id}]									len() = 5

SyncVars Update by Client/Server -> Payload:
[SYNCVAR_UPDATE					| (n1){ (2)(id)			| (2)(SyncVar Length) 		| (n2)(SyncVar Data) }]						len() = 1+n*(4+n2)
//1 Int64 --> 13 byte

SyncVar Delete -> 					Payload:
[SYNCVAR_DELETION				| (2){id} 				| (n){name}]															len() = 2+n
*/
//

func GetClientManager(c *Client) (cm *ClientManager) {
	cm = &ClientManager{Client: c}
	cm.SyncvarsByID = make(map[int]SyncVar)
	cm.SyncvarsByName = make(map[string]SyncVar)
	cm.NameToID = make(map[string]int)
	c.InputHandler = cm.receive
	return
}

type ClientManager struct {
	Client         *Client
	SyncvarsByID   map[int]SyncVar
	SyncvarsByName map[string]SyncVar
	NameToID       map[string]int

	InputHandler func(mt int, msg []byte, err error, c *Client) (alive bool)
}

func (m *ClientManager) RegisterSyncVar(sv SyncVar, name string) {
	if m.SyncvarsByID == nil {
		m.SyncvarsByID = make(map[int]SyncVar)
	}
	if m.SyncvarsByName == nil {
		m.SyncvarsByName = make(map[string]SyncVar)
	}
	printLogF(".....Client: Requesting SyncVar with name='%s', type=%v , initiated by self=%s", name, sv.Type(), m.Client.LocalAddr().String())
	m.SyncvarsByName[name] = sv
	m.Client.Send(append([]byte{SYNCVAR_REGISTRY, sv.Type()}, []byte(name)...))
}
func (m *ClientManager) UpdateSyncVars() {
	var_data := []byte{SYNCVAR_UPDATE}
	for id, sv := range m.SyncvarsByID {
		if sv.IsDirty() {
			syncDat := sv.GetData()
			printLogF(".....Client: Updating SyncVar with ID=%v: len(dat)=%v, initiated by self=%s", id, len(syncDat), m.Client.LocalAddr().String())
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			payload := append(cmp.Int16ToBytes(int16(id)), data...)
			var_data = append(var_data, payload...)
		}
	}
	if len(var_data) > 1 {
		m.Client.Send(var_data)
	}
}
func (m *ClientManager) DeleteSyncVar(name string) {
	id := m.NameToID[name]
	printLogF(".....Client: Deleting SyncVar with ID=%v, name='%s' , initiated by self=%s", id, name, m.Client.LocalAddr().String())
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
	} else if input[0] == SYNCVAR_REGISTRY {
		l := input[2]
		t := input[1]
		name := string(input[3 : 3+l])
		id := int(cmp.BytesToInt16(input[3+l:]))
		printLogF(".....Client: Creating SyncVar with ID=%v, name='%s' , initiated by server=%s", id, name, m.Client.RemoteAddr().String())
		m.SyncvarsByName[name] = GetSyncVarOfType(t)
		m.SyncvarsByID[id] = m.SyncvarsByName[name]
		m.NameToID[name] = id
	} else if input[0] == SYNCVAR_UPDATE {
		m.onSyncVarUpdateC(input[1:])
	} else if input[0] == SYNCVAR_DELETION {
		id := int(cmp.BytesToInt16(input[1:3]))
		name := string(input[3:])
		m.deleteSyncVarLocal(name, id)
		printLogF(".....Client: Deleting SyncVar with ID=%v, name='%s' , initiated by server=%s", id, name, m.Client.RemoteAddr().String())
	}
	if m.InputHandler != nil {
		return m.InputHandler(mt, input, err, c)
	}
	return true
}
func (m *ClientManager) processRegisterVarConfirm(data []byte) {
	l := data[0]
	data = data[1:]
	name := string(data[:l])
	id := int(cmp.BytesToInt16(data[l:]))
	m.SyncvarsByID[id] = m.SyncvarsByName[name]
	m.NameToID[name] = id
	printLogF(".....Client: Creating SyncVar with ID=%v, name='%s' , confirmed by server=%s", id, name, m.Client.RemoteAddr().String())
}
func (m *ClientManager) onSyncVarUpdateC(data []byte) {
	for true {
		id := int(cmp.BytesToInt16(data[:2]))
		l := cmp.BytesToInt16(data[2:4])
		dat := data[4 : 4+l]
		printLogF(".....Client: Updating SyncVar with ID=%v: len(dat)=%v, initiated by server=%s", id, l, m.Client.RemoteAddr().String())
		data = data[4+l:]
		m.SyncvarsByID[id].SetData(dat)
		if len(data) <= 0 {
			break
		}
	}
}
