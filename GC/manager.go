package GC

import (
	cmp "marvin/GraphEng/Compression"
)

type ClientManager struct {
	Client			*Client
	SyncvarsByID	map[int]SyncVar
	SyncvarsByName	map[string]SyncVar

}
func (m *ClientManager) Receive(input []byte) {
	if input[0] == SYNCVAR_REGISTRY_RESPONSE {
		m.processRegisterVarRespose(input[1:])
	}
}
func (m *ClientManager) RegisterVar(sv SyncVar, name string) {
	if m.SyncvarsByID == nil {
		m.SyncvarsByID = make(map[int]SyncVar)
	}
	if m.SyncvarsByName == nil {
		m.SyncvarsByName = make(map[string]SyncVar)
	}
	m.SyncvarsByName[name] = sv
	m.Client.Send(append([]byte{SYNCVAR_REGISTRY, sv.Type()}, []byte(name)...))
}
func (m *ClientManager) processRegisterVarRespose(data []byte) {
	idx := data[0]; data = data[1:]
	name := string(data[:idx])
	id := cmp.BytesToInt64(data[idx:])
	m.SyncvarsByID[int(id)] = m.SyncvarsByName[name]
}



//For every connected client one handler with a map of syncvars
/**
type ServerManager struct {
	Server   *Server
	SyncvarsByID	map[int]SyncVar
	SyncvarsByName	map[string]SyncVar
	
	idCounter int
}
func (m *ServerManager) Receive(input []byte) {
	if input[0] == SYNCVAR_REGISTRY {
		m.processRegistry(input[1:])
	}
}
func (m *ServerManager) RegisterVar(sv SyncVar, name string, clients ...int) {
	m.SyncvarsByName[name] = sv
	m.Server.SendToMultiple(append([]byte{SYNCVAR_REGISTRY, sv.Type()}, []byte(name)...), clients...)
}
func (m *ServerManager) processRegistry(data []byte) {
	t := data[0]
	name := string(data[1:])
	id := m.idCounter
	m.SyncvarsByID[id] = GetSyncVarOfType(t)
	m.SyncvarsByName[name] = m.SyncvarsByID[id]
	
	
	m.Server
	
	m.idCounter ++
}
func (m *ServerManager) Init() {
	if m.SyncvarsByID == nil {
		m.SyncvarsByID = make(map[int]SyncVar)
	}
	if m.SyncvarsByName == nil {
		m.SyncvarsByName = make(map[string]SyncVar)
	}
	m.idCounter = 0
}
**/