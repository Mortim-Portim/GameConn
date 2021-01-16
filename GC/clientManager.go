package GC

import (
	cmp "github.com/mortim-portim/GraphEng/Compression"
	//"sync"
)

/**
Protocoll:
Client registers SyncVar -> 		Payload:
[SYNCVAR_REGISTRY 				| SyncVar.Type() 		| (2){AccessID}]														len() = 4

-> Server sends Confirmation -> 	Payload:
[SYNCVAR_REGISTRY_CONFIRMATION 	| (2){AccessID}] 		| (2){id}]																len() = 5

Client registers multiple SyncVars->Payload:
[SYNCVAR_M_REGISTRY 			| (3){SyncVar.Type() 	| (2){AccessID}}]														len() = 1+3*n

-> Server sends Confirmations ->    Payload:
[SYNCVAR_M_REGISTRY_CONFIRMATION| (4){(2){AccessID}] 	| (2){id}}]																len() = 1+4*n

Server registers SyncVar -> 		Payload:
[SYNCVAR_REGISTRY 				| SyncVar.Type() 		| (2){AccessID}				| (2){id}]									len() = 6

Server registers multiple SyncVars->Payload:
[SYNCVAR_M_REGISTRY 			| (5){SyncVar.Type() 	| (2){AccessID}				| (2){id}}]									len() = 1+5*n

SyncVars Update by Client/Server -> Payload:
[SYNCVAR_UPDATE					| (n1){ (2)(id)			| (2)(SyncVar Length) 		| (n2)(SyncVar Data) }]						len() = 1+n1*(4+n2)
//1 Int64 --> 13 byte

SyncVar Delete -> 					Payload:
[SYNCVAR_DELETION				| (2){id} 				| (2){AccessID}]														len() = 5
**/

func GetClientManager(c *Client) (cm *ClientManager) {
	cm = &ClientManager{Client:c}
	cm.SyncvarsByID = 	make(map[int]SyncVar)
	cm.SyncvarsByACID = make(map[int]SyncVar)
	cm.ACIDToID	=		make(map[int]int)
	cm.SyncVarOnChange = make(map[int]func(SyncVar, int))
	c.InputHandler = 	cm.receive
	return
}
type ClientManager struct {
	Client			*Client
	SyncvarsByID	map[int]SyncVar
	SyncvarsByACID	map[int]SyncVar
	ACIDToID		map[int]int
	SyncVarOnChange map[int]func(SyncVar, int)
	
	InputHandler  func(mt int, msg []byte, err error, c *Client) (alive bool)
	
	//SV_ID, SV_ACID, ACID_ID, Chng sync.Mutex
}
func (m *ClientManager) RegisterSyncVar(sv SyncVar, ACID int) {
	printLogF(".....Client: Requesting SyncVar with ACID='%v', type=%v , initiated by self=%s", ACID, sv.Type(), m.Client.LocalAddr().String())
	sv.IsRegisteredTo(1)
	m.SyncvarsByACID[ACID] = sv
	m.Client.Send(append([]byte{SYNCVAR_REGISTRY, sv.Type()}, cmp.Int16ToBytes(int16(ACID))...))
}
func (m *ClientManager) RegisterSyncVars(svs map[int]SyncVar) {
	data := []byte{SYNCVAR_M_REGISTRY}
	for ACID,sv := range(svs) {
		//printLogF(".....Client: MRequesting SyncVar with ACID='%v', type=%v , initiated by self=%s", ACID, sv.Type(), m.Client.LocalAddr().String())
		sv.IsRegisteredTo(1)
		m.SyncvarsByACID[ACID] = sv
		data = append(data, sv.Type())
		data = append(data, cmp.Int16ToBytes(int16(ACID))...)
	}
	printLogF(".....Client: MRequesting %v SyncVars, initiated by self=%s", len(svs), m.Client.LocalAddr().String())
	m.Client.Send(data)
}
func (m *ClientManager) RegisterOnChangeFunc(ACID int, fnc func(SyncVar, int)) {
	id, ok := m.ACIDToID[ACID]
	if ok {
		m.SyncVarOnChange[id] = fnc
	}
}

func (m *ClientManager) UpdateSyncVarsWithACIDs(ACIDs ...int) (uc int) {
	lastACID := -1
	var_data := []byte{SYNCVAR_UPDATE}
	for ACID,sv := range(m.SyncvarsByACID) {
		if sv.IsDirty() && containsI(ACIDs, ACID) {
			uc ++
			lastACID = ACID
			
			printLogF(".....Client: Updating SyncVar with ACID: %v of type %v: %v, initiated by self=%s", lastACID, sv.Type(), sv, m.Client.LocalAddr().String())
			
			syncDat := sv.GetData()
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			payload := append(cmp.Int16ToBytes(int16(m.ACIDToID[ACID])), data...)
			var_data = append(var_data, payload...)
		}
	}
	if uc > 1 {
		printLogF(".....Client: Updating %v SyncVars, initiated by self=%s", uc, m.Client.LocalAddr().String())
	}else if uc == 1 {
		//sv := m.SyncvarsByACID[lastACID]
		//printLogF(".....Client: Updating SyncVar with ACID: %v of type %v: %v, initiated by self=%s", lastACID, sv.Type(), sv, m.Client.LocalAddr().String())
	}
	if len(var_data) > 1 {
		m.Client.Send(var_data)
	}
	return
}
func (m *ClientManager) UpdateSyncVars() (uc int) {
	lastID := -1
	var_data := []byte{SYNCVAR_UPDATE}
	for id,sv := range(m.SyncvarsByID) {
		if sv.IsDirty() {
			uc ++
			lastID = id
			syncDat := sv.GetData()
			//printLogF(".....Client: Updating SyncVar with ID=%v: len(dat)=%v, initiated by self=%s", id, len(syncDat), m.Client.LocalAddr().String())
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			payload := append(cmp.Int16ToBytes(int16(id)), data...)
			var_data = append(var_data, payload...)
		}
	}
	if uc > 1 {
		printLogF(".....Client: Updating %v SyncVars, initiated by self=%s", uc, m.Client.LocalAddr().String())
	}else if uc == 1 {
		sv := m.SyncvarsByID[lastID]
		printLogF(".....Client: Updating SyncVar with ID: %v of type %v: %v, initiated by self=%s", lastID, sv.Type(), sv, m.Client.LocalAddr().String())
	}
	if len(var_data) > 1 {
		m.Client.Send(var_data)
	}
	return
}
func (m *ClientManager) DeleteSyncVar(ACID int) {
	id := m.ACIDToID[ACID]
	printLogF(".....Client: Deleting SyncVar with ID=%v, ACID='%v' , initiated by self=%s", id, ACID, m.Client.LocalAddr().String())
	m.deleteSyncVarLocal(ACID, id)
	m.deleteSyncVarRemote(ACID, id)
}
func (m *ClientManager) deleteSyncVarLocal(ACID int, id int) {
	delete(m.SyncvarsByID, id)
	delete(m.SyncvarsByACID, ACID)
	delete(m.ACIDToID, ACID)
}
func (m *ClientManager) deleteSyncVarRemote(ACID int, id int) {
	data := append(cmp.Int16ToBytes(int16(id)), cmp.Int16ToBytes(int16(ACID))...)
	m.Client.Send(append([]byte{SYNCVAR_DELETION}, data...))
}
func (m *ClientManager) receive(mt int, input []byte, err error, c *Client) bool {
	if err != nil {
		return false
	}
	if input[0] == SYNCVAR_REGISTRY_CONFIRMATION {
		m.processRegisterVarConfirm(input[1:])
	}else if input[0] == SYNCVAR_M_REGISTRY_CONFIRMATION {
		m.processRegisterVarsConfirm(input[1:])
	}else if input[0] == SYNCVAR_REGISTRY {
		t := input[1]
		ACID := int(cmp.BytesToInt16(input[2:4]))
		id := int(cmp.BytesToInt16(input[4:6]))
		printLogF(".....Client: Creating SyncVar with ID=%v, ACID='%v' , initiated by server=%s", id, ACID, m.Client.RemoteAddr().String())
		m.SyncvarsByACID[ACID] = GetSyncVarOfType(t)
		m.SyncvarsByID[id] = m.SyncvarsByACID[ACID]
		m.ACIDToID[ACID] = id
	}else if input[0] == SYNCVAR_M_REGISTRY {
		input = input[1:]
		for i := 0; i < len(input); i += 5 {
			t := input[i]
			ACID := int(cmp.BytesToInt16(input[i+1:i+3]))
			id := int(cmp.BytesToInt16(input[i+3:i+5]))
			//printLogF(".....Client: MCreating SyncVar with ID=%v, ACID='%v' , initiated by server=%s", id, ACID, m.Client.RemoteAddr().String())
			m.SyncvarsByACID[ACID] = GetSyncVarOfType(t)
			m.SyncvarsByID[id] = m.SyncvarsByACID[ACID]
			m.ACIDToID[ACID] = id
		}
		printLogF(".....Client: MCreating %v SyncVars, initiated by server=%s", len(input)/5, m.Client.RemoteAddr().String())
	}else if input[0] == SYNCVAR_UPDATE {
		m.onSyncVarUpdateC(input[1:])
	}else if input[0] == SYNCVAR_DELETION {
		id := int(cmp.BytesToInt16(input[1:3]))
		ACID := int(cmp.BytesToInt16(input[3:5]))
		m.deleteSyncVarLocal(ACID, id)
		printLogF(".....Client: Deleting SyncVar with ID=%v, ACID='%v' , initiated by server=%s", id, ACID, m.Client.RemoteAddr().String())
	}
	if m.InputHandler != nil {
		return m.InputHandler(mt, input, err, c)
	}
	return true
}
func (m *ClientManager) processRegisterVarConfirm(data []byte) {
	ACID := int(cmp.BytesToInt16(data[0:2]))
	id := int(cmp.BytesToInt16(data[2:4]))
	m.SyncvarsByID[id] = m.SyncvarsByACID[ACID]
	m.ACIDToID[ACID] = id
	printLogF(".....Client: Creating SyncVar with ID=%v, ACID='%v' , confirmed by server=%s", id, ACID, m.Client.RemoteAddr().String())
}
func (m *ClientManager) processRegisterVarsConfirm(data []byte) {
	for i := 0; i < len(data); i += 4 {
		ACID := int(cmp.BytesToInt16(data[i:i+2]))
		id := int(cmp.BytesToInt16(data[i+2:i+4]))
		m.SyncvarsByID[id] = m.SyncvarsByACID[ACID]
		m.ACIDToID[ACID] = id
		//printLogF(".....Client: MCreating SyncVar with ID=%v, ACID='%v' , confirmed by server=%s", id, ACID, m.Client.RemoteAddr().String())
	}
	printLogF(".....Client: MCreating %v SyncVars, confirmed by server=%s", len(data)/4, m.Client.RemoteAddr().String())
}
func (m *ClientManager) onSyncVarUpdateC(data []byte) {
	num := 0
	for true {
		num ++
		id := int(cmp.BytesToInt16(data[:2]))
		l := cmp.BytesToInt16(data[2:4])
		dat := data[4:4+l]
		//printLogF(".....Client: Updating SyncVar with ID=%v: len(dat)=%v, initiated by server=%s", id, l, m.Client.RemoteAddr().String())
		data = data[4+l:]
		m.SyncvarsByID[id].SetData(dat)
		if fnc, ok := m.SyncVarOnChange[id]; ok {
			fnc(m.SyncvarsByID[id], id)
		}
		if len(data) <= 0 {
			break
		}
	}
	printLogF(".....Client: Updating %v SyncVars, initiated by server=%s", num, m.Client.RemoteAddr().String())
}

//Returns true if e is in s
func containsI(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}