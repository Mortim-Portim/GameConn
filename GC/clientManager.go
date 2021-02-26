package GC

import (
	"fmt"
	"sync"

	cmp "github.com/mortim-portim/GraphEng/compression"
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
[SYNCVAR_UPDATE					| (n1){ (2)(id)			| (4)(SyncVar Length) 		| (n2)(SyncVar Data) }]						len() = 1+n1*(6+n2)
//1 Int64 --> 13 byte

SyncVar Delete -> 					Payload:
[SYNCVAR_DELETION				| (2){id} 				| (2){AccessID}]														len() = 5
**/

func GetClientManager(c *Client) (cm *ClientManager) {
	cm = &ClientManager{Client: c}
	cm.reset()
	return
}
func (m *ClientManager) reset() {
	m.SyncvarsByID = make(map[int]SyncVar)
	m.SyncvarsByACID = make(map[int]SyncVar)
	m.ACIDToID = make(map[int]int)
	m.SyncVarOnChange = make(map[int]func(SyncVar, int))
	m.Client.InputHandler = m.receive
	m.Client.OnCloseConnection = m.ClientConnClose
}
func (m *ClientManager) ClientConnClose() {
	m.reset()
	if m.OnCloseConnection != nil {
		m.OnCloseConnection()
	}
}

type ClientManager struct {
	Client          *Client
	SyncvarsByID    map[int]SyncVar
	SyncvarsByACID  map[int]SyncVar
	ACIDToID        map[int]int
	SyncVarOnChange map[int]func(SyncVar, int)

	SV_ID_L, SV_ACID_L, ACID_ID_L, SV_chng sync.Mutex

	InputHandler      func(mt int, msg []byte, err error, c *Client) (alive bool)
	OnCloseConnection func()
	//SV_ID, SV_ACID, ACID_ID, Chng sync.Mutex
}

//you may wait for confirmation
func (m *ClientManager) RegisterSyncVarNormal(sv SyncVar, ACID int) {
	m.Client.SendBuffered(m.registerSyncVar(sv, ACID))
}
func (m *ClientManager) RegisterSyncVarBuffered(sv SyncVar, ACID int) {
	m.Client.SendNormal(m.registerSyncVar(sv, ACID))
}
func (m *ClientManager) registerSyncVar(sv SyncVar, ACID int) []byte {
	printLogF(3, ".....Client: Requesting SyncVar with ACID='%v', type=%v , initiated by self=%s", ACID, sv.Type(), m.Client.LocalAddr().String())
	sv.IsRegisteredTo(1)
	m.SV_ACID_L.Lock()
	m.SyncvarsByACID[ACID] = sv
	m.SV_ACID_L.Unlock()
	return append([]byte{SYNCVAR_REGISTRY, sv.Type()}, cmp.Int16ToBytes(int16(ACID))...)
}

//you may wait for confirmation
func (m *ClientManager) RegisterSyncVarsNormal(svs map[int]SyncVar) {
	m.Client.SendNormal(m.registerSyncVars(svs))
}
func (m *ClientManager) RegisterSyncVarsBuffered(svs map[int]SyncVar) {
	m.Client.SendBuffered(m.registerSyncVars(svs))
}
func (m *ClientManager) registerSyncVars(svs map[int]SyncVar) []byte {
	data := []byte{SYNCVAR_M_REGISTRY}
	for ACID, sv := range svs {
		//printLogF(".....Client: MRequesting SyncVar with ACID='%v', type=%v , initiated by self=%s", ACID, sv.Type(), m.Client.LocalAddr().String())
		sv.IsRegisteredTo(1)
		m.SV_ACID_L.Lock()
		m.SyncvarsByACID[ACID] = sv
		m.SV_ACID_L.Unlock()
		data = append(data, sv.Type())
		data = append(data, cmp.Int16ToBytes(int16(ACID))...)
	}
	printLogF(3, ".....Client: MRequesting %v SyncVars, initiated by self=%s", len(svs), m.Client.LocalAddr().String())
	return data
}
func (m *ClientManager) RegisterOnChangeFunc(ACID int, fnc func(SyncVar, int)) {
	m.ACID_ID_L.Lock()
	id, ok := m.ACIDToID[ACID]
	m.ACID_ID_L.Unlock()
	if ok {
		m.SV_chng.Lock()
		m.SyncVarOnChange[id] = fnc
		m.SV_chng.Unlock()
	}
}

//you may wait for confirmation
func (m *ClientManager) UpdateSyncVarsWithACIDsNormal(ACIDs ...int) (uc int) {
	var_data, count := m.updateSyncVarsByACIDAndReturnData(false, ACIDs...)
	uc = count
	if uc > 1 {
		printLogF(2, ".....Client: Updating %v SyncVars, initiated by self=%s", uc, m.Client.LocalAddr().String())
	}
	if len(var_data) > 1 {
		m.Client.SendNormal(var_data)
	}
	return
}
func (m *ClientManager) UpdateSyncVarsWithACIDsBuffered(ACIDs ...int) (uc int) {
	var_data, count := m.updateSyncVarsByACIDAndReturnData(false, ACIDs...)
	uc = count
	if uc > 1 {
		printLogF(2, ".....Client: Updating %v SyncVars, initiated by self=%s", uc, m.Client.LocalAddr().String())
	}
	if len(var_data) > 1 {
		m.Client.SendBuffered(var_data)
	}
	return
}

//you may wait for confirmation
func (m *ClientManager) UpdateSyncVarsNormal() (uc int) {
	var_data, count := m.updateSyncVarsByIDAndReturnData(true)
	uc = count
	if uc > 1 {
		printLogF(2, ".....Client: Updating %v SyncVars, initiated by self=%s", uc, m.Client.LocalAddr().String())
	}
	if len(var_data) > 1 {
		m.Client.SendNormal(var_data)
	}
	return
}
func (m *ClientManager) UpdateSyncVarsBuffered() (uc int) {
	var_data, count := m.updateSyncVarsByIDAndReturnData(true)
	uc = count
	if uc > 1 {
		printLogF(2, ".....Client: Updating %v SyncVars, initiated by self=%s", uc, m.Client.LocalAddr().String())
	}
	if len(var_data) > 1 {
		m.Client.SendBuffered(var_data)
	}
	return
}
func (m *ClientManager) updateSyncVarsByACIDAndReturnData(all bool, ACIDs ...int) (var_data []byte, uc int) {
	var_data = []byte{SYNCVAR_UPDATE}
	m.SV_ACID_L.Lock()
	for ACID, sv := range m.SyncvarsByACID {
		if sv.IsDirty() && (all || containsI(ACIDs, ACID)) {
			uc++
			syncDat := sv.GetData()
			data := append(cmp.Uint32ToBytes(uint32(len(syncDat))), syncDat...)
			m.ACID_ID_L.Lock()
			payload := append(cmp.Int16ToBytes(int16(m.ACIDToID[ACID])), data...)
			m.ACID_ID_L.Unlock()
			var_data = append(var_data, payload...)
			//printLogF(2, ".....Client: Updating SyncVar with ACID: %v of type %v: %v, initiated by self=%s", lastACID, sv.Type(), sv, m.Client.LocalAddr().String())
		}
	}
	m.SV_ACID_L.Unlock()
	return
}
func (m *ClientManager) updateSyncVarsByIDAndReturnData(all bool, IDs ...int) (var_data []byte, uc int) {
	var_data = []byte{SYNCVAR_UPDATE}
	m.SV_ID_L.Lock()
	for id, sv := range m.SyncvarsByID {
		if sv.IsDirty() && (all || containsI(IDs, id)) {
			uc++
			syncDat := sv.GetData()
			//printLogF(".....Client: Updating SyncVar with ID=%v: len(dat)=%v, initiated by self=%s", id, len(syncDat), m.Client.LocalAddr().String())
			data := append(cmp.Uint32ToBytes(uint32(len(syncDat))), syncDat...)
			payload := append(cmp.Int16ToBytes(int16(id)), data...)
			var_data = append(var_data, payload...)
		}
	}
	m.SV_ID_L.Unlock()
	return
}

//you may wait for confirmation
func (m *ClientManager) DeleteSyncVarNormal(ACID int) {
	m.ACID_ID_L.Lock()
	id := m.ACIDToID[ACID]
	m.ACID_ID_L.Unlock()
	printLogF(3, ".....Client: Deleting SyncVar with ID=%v, ACID='%v' , initiated by self=%s", id, ACID, m.Client.LocalAddr().String())
	m.deleteSyncVarLocal(ACID, id)
	m.deleteSyncVarRemoteNormal(ACID, id)
}
func (m *ClientManager) DeleteSyncVarBuffered(ACID int) {
	m.ACID_ID_L.Lock()
	id := m.ACIDToID[ACID]
	m.ACID_ID_L.Unlock()
	printLogF(3, ".....Client: Deleting SyncVar with ID=%v, ACID='%v' , initiated by self=%s", id, ACID, m.Client.LocalAddr().String())
	m.deleteSyncVarLocal(ACID, id)
	m.deleteSyncVarRemoteBuffered(ACID, id)
}
func (m *ClientManager) deleteSyncVarLocal(ACID int, id int) {
	m.SV_ID_L.Lock()
	delete(m.SyncvarsByID, id)
	m.SV_ID_L.Unlock()
	m.SV_ACID_L.Lock()
	delete(m.SyncvarsByACID, ACID)
	m.SV_ACID_L.Unlock()
	m.ACID_ID_L.Lock()
	delete(m.ACIDToID, ACID)
	m.ACID_ID_L.Unlock()
}
func (m *ClientManager) deleteSyncVarRemoteNormal(ACID int, id int) {
	data := append(cmp.Int16ToBytes(int16(id)), cmp.Int16ToBytes(int16(ACID))...)
	m.Client.SendNormal(append([]byte{SYNCVAR_DELETION}, data...))
}
func (m *ClientManager) deleteSyncVarRemoteBuffered(ACID int, id int) {
	data := append(cmp.Int16ToBytes(int16(id)), cmp.Int16ToBytes(int16(ACID))...)
	m.Client.SendBuffered(append([]byte{SYNCVAR_DELETION}, data...))
}
func (m *ClientManager) receive(mt int, input []byte, err error, c *Client) (alive bool) {
	alive = true
	if err != nil {
		alive = false
	}
	if input[0] == SYNCVAR_REGISTRY_CONFIRMATION {
		m.processRegisterVarConfirm(input[1:])
	} else if input[0] == SYNCVAR_M_REGISTRY_CONFIRMATION {
		m.processRegisterVarsConfirm(input[1:])
	} else if input[0] == SYNCVAR_REGISTRY {
		t := input[1]
		ACID := int(cmp.BytesToInt16(input[2:4]))
		id := int(cmp.BytesToInt16(input[4:6]))
		printLogF(3, ".....Client: Creating SyncVar with ID=%v, ACID='%v' , initiated by server=%s", id, ACID, m.Client.RemoteAddr().String())
		m.SV_ACID_L.Lock()
		m.SyncvarsByACID[ACID] = GetSyncVarOfType(t)
		m.SV_ID_L.Lock()
		m.SyncvarsByID[id] = m.SyncvarsByACID[ACID]
		m.SV_ACID_L.Unlock()
		m.SV_ID_L.Unlock()
		m.ACID_ID_L.Lock()
		m.ACIDToID[ACID] = id
		m.ACID_ID_L.Unlock()
	} else if input[0] == SYNCVAR_M_REGISTRY {
		input = input[1:]
		for i := 0; i < len(input); i += 5 {
			t := input[i]
			ACID := int(cmp.BytesToInt16(input[i+1 : i+3]))
			id := int(cmp.BytesToInt16(input[i+3 : i+5]))
			//printLogF(".....Client: MCreating SyncVar with ID=%v, ACID='%v' , initiated by server=%s", id, ACID, m.Client.RemoteAddr().String())
			m.SV_ACID_L.Lock()
			m.SyncvarsByACID[ACID] = GetSyncVarOfType(t)
			m.SV_ID_L.Lock()
			m.SyncvarsByID[id] = m.SyncvarsByACID[ACID]
			m.SV_ACID_L.Unlock()
			m.SV_ID_L.Unlock()
			m.ACID_ID_L.Lock()
			m.ACIDToID[ACID] = id
			m.ACID_ID_L.Unlock()
		}
		printLogF(3, ".....Client: MCreating %v SyncVars, initiated by server=%s", len(input)/5, m.Client.RemoteAddr().String())
	} else if input[0] == SYNCVAR_UPDATE {
		m.onSyncVarUpdateC(input[1:])
	} else if input[0] == SYNCVAR_DELETION {
		id := int(cmp.BytesToInt16(input[1:3]))
		ACID := int(cmp.BytesToInt16(input[3:5]))
		m.deleteSyncVarLocal(ACID, id)
		printLogF(3, ".....Client: Deleting SyncVar with ID=%v, ACID='%v' , initiated by server=%s", id, ACID, m.Client.RemoteAddr().String())
	}
	if m.InputHandler != nil {
		return m.InputHandler(mt, input, err, c)
	}
	return
}
func (m *ClientManager) processRegisterVarConfirm(data []byte) {
	ACID := int(cmp.BytesToInt16(data[0:2]))
	id := int(cmp.BytesToInt16(data[2:4]))
	m.SV_ID_L.Lock()
	m.SV_ACID_L.Lock()
	m.SyncvarsByID[id] = m.SyncvarsByACID[ACID]
	m.SV_ACID_L.Unlock()
	m.SV_ID_L.Unlock()
	m.ACID_ID_L.Lock()
	m.ACIDToID[ACID] = id
	m.ACID_ID_L.Unlock()
	printLogF(3, ".....Client: Creating SyncVar with ID=%v, ACID='%v' , confirmed by server=%s", id, ACID, m.Client.RemoteAddr().String())
}
func (m *ClientManager) processRegisterVarsConfirm(data []byte) {
	for i := 0; i < len(data); i += 4 {
		ACID := int(cmp.BytesToInt16(data[i : i+2]))
		id := int(cmp.BytesToInt16(data[i+2 : i+4]))
		m.SV_ID_L.Lock()
		m.SV_ACID_L.Lock()
		m.SyncvarsByID[id] = m.SyncvarsByACID[ACID]
		m.SV_ACID_L.Unlock()
		m.SV_ID_L.Unlock()
		m.ACID_ID_L.Lock()
		m.ACIDToID[ACID] = id
		m.ACID_ID_L.Unlock()
		//printLogF(".....Client: MCreating SyncVar with ID=%v, ACID='%v' , confirmed by server=%s", id, ACID, m.Client.RemoteAddr().String())
	}
	printLogF(3, ".....Client: MCreating %v SyncVars, confirmed by server=%s", len(data)/4, m.Client.RemoteAddr().String())
}
func (m *ClientManager) onSyncVarUpdateC(testData []byte) {
	allSyncVarsData := make([]byte, len(testData))
	copy(allSyncVarsData, testData)
	num := 0
	printLogF(1, "Updating syncVars from data of len(%v): %v\n", len(allSyncVarsData), allSyncVarsData)
	for true {
		num++
		printLogF(1, "allSyncVarsData1: %v\n", allSyncVarsData)
		id := int(cmp.BytesToInt16(allSyncVarsData[:2]))
		printLogF(1, "ID: %v\n", id)
		l := cmp.BytesToUint32(allSyncVarsData[2:6])
		printLogF(1, "len: %v\n", l)
		svData := allSyncVarsData[6 : 6+l]
		printLogF(1, "svData: %v\n", svData)
		//printLogF(".....Client: Updating SyncVar with ID=%v: len(dat)=%v, initiated by server=%s", id, l, m.Client.RemoteAddr().String())
		printLogF(1, "allSyncVarsData2: %v\n", allSyncVarsData)
		allSyncVarsData = allSyncVarsData[6+l:]
		printLogF(1, "allSyncVarsData3: %v\n", allSyncVarsData)
		m.SV_ID_L.Lock()
		sv, ok := m.SyncvarsByID[id]
		m.SV_ID_L.Unlock()
		printLogF(1, "allSyncVarsData4: %v\n", allSyncVarsData)
		if !ok {
			panic(fmt.Sprintf("%v not in map %v, data: %v, l: %v\n", id, m.SyncvarsByID, svData, l))
		}
		sv.SetData(svData)
		printLogF(1, "allSyncVarsData5: %v\n", allSyncVarsData)
		if fnc, ok := m.SyncVarOnChange[id]; ok {
			printLogF(1, "allSyncVarsData5.1: %v\n", allSyncVarsData)
			printLogF(1, "SyncVar: %v\n", sv.Type())
			fnc(sv, id)
			printLogF(1, "allSyncVarsData5.2: %v\n", allSyncVarsData)
		}
		printLogF(1, "allSyncVarsData6: %v\n", allSyncVarsData)
		if len(allSyncVarsData) <= 0 {
			break
		}
		printLogF(1, "allSyncVarsData7: %v\n", allSyncVarsData)
	}
	printLogF(2, ".....Client: Updating %v SyncVars, initiated by server=%s", num, m.Client.RemoteAddr().String())
}

//9 128 47 0 0 0 1 45 7 33 128 0 124 93 40 0 42 236 192 191 178 110 35 194 133 53 195 191 29 67 188 52 69 102 30 64 89 49 52 64 251 217 29 64 1 0 5 8 128 7 128 25 245 13 128 13 0 0 0 1 11 0 162 141 162 12 178 6 19 64 44 0
//9 128 47 0 0 0 1 45 7 33 128 0 124 93 40 0 42 236 192 191 178 110 35 194 133 53 195 191 29 67 188 52 69 102 30 64 89 49 52 64 251 217 29 64 1 0 5 8 128 7 128 25 245 13 128 13 0 0 0 1 11 0 162 141 162 12 178 6 19 64 44 0
//9 128 47 0 0 0 1 45 7 33 128 0 124 93 40 0 42 236 192 191 178 110 35 194 133 53 195 191 29 67 188 52 69 102 30 64 89 49 52 64 251 217 29 64 1 0 5 8 128 7 128 25 245 13 128 13 0 0 0 1 11 0 162 141 162 12 178 6 19 64 44 0
//1 45 7 33 128 0 124 93 40 0 42 236 192 191 178 110 35 194 133 53 195 191 29 67 188 52 69 102 30 64 89 49 52 64 251 217 29 64 1 0 5 8 128 7 128 25 245 13 128 13 0 0 0 1 11 0 162 141 162 12 178 6 19 64 44 0
