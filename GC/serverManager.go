package GC

import (
	ws "github.com/gorilla/websocket"
	cmp "github.com/mortim-portim/GraphEng/Compression"
)

type Handler interface {
	RegisterSyncVar(sv SyncVar, ACID int)
	RegisterSyncVars(svs []SyncVar, ACIDs ...int)
	RegisterOnChangeFunc(ACID int, fnc func(SyncVar, int))
	UpdateSyncVars() int
	DeleteSyncVar(ACID int)
}

func GetNewClientHandler(s *Server, cn *ws.Conn) (ch *ClientHandler) {
	ch = &ClientHandler{Server:s, Conn:cn}
	ch.idCounter = 0
	ch.SyncvarsByID = 	make(map[int]SyncVar)
	ch.SyncvarsByACID = make(map[int]SyncVar)
	ch.ACIDToID	=		make(map[int]int)
	ch.SyncVarOnChange = make(map[int]func(SyncVar, int))
	return
}
type ClientHandler struct {
	Server 	*Server
	Conn	*ws.Conn
	SyncvarsByID	map[int]SyncVar
	SyncvarsByACID	map[int]SyncVar
	ACIDToID		map[int]int
	SyncVarOnChange map[int]func(SyncVar, int)
	idCounter int
}
func (ch *ClientHandler) RegisterSyncVar(sv SyncVar, ACID int) {
	printLogF("#####Server: Creating SyncVar with ID=%v, ACID='%v', type=%v , initiated by self=%s", ch.idCounter, ACID, sv.Type(), ch.Conn.LocalAddr().String())
	ch.SyncvarsByACID[ACID] 		= sv
	ch.SyncvarsByID[ch.idCounter] 	= sv
	ch.ACIDToID[ACID] 				= ch.idCounter
	resp := append(append([]byte{SYNCVAR_REGISTRY, sv.Type()}, cmp.Int16ToBytes(int16(ACID))...), cmp.Int16ToBytes(int16(ch.idCounter))...)
	ch.Server.Send(resp, ch.Server.ConnToIdx[ch.Conn])
	ch.idCounter ++
}
func (ch *ClientHandler) RegisterSyncVars(svs []SyncVar, ACIDs ...int) {
	data := []byte{SYNCVAR_M_REGISTRY}
	for i,sv := range(svs) {
		ACID := ACIDs[i]
		printLogF("#####Server: MCreating SyncVar with ID=%v, ACID='%v', type=%v , initiated by self=%s", ch.idCounter, ACID, sv.Type(), ch.Conn.LocalAddr().String())
		ch.SyncvarsByACID[ACID] 		= sv
		ch.SyncvarsByID[ch.idCounter] 	= sv
		ch.ACIDToID[ACID] 				= ch.idCounter
		data = append(data, sv.Type())
		data = append(data, cmp.Int16ToBytes(int16(ACID))...)
		data = append(data, cmp.Int16ToBytes(int16(ch.idCounter))...)
		ch.idCounter ++
	}
	ch.Server.Send(data, ch.Server.ConnToIdx[ch.Conn])
}
func (ch *ClientHandler) RegisterOnChangeFunc(ACID int, fnc func(SyncVar, int)) {
	id, ok := ch.ACIDToID[ACID]
	if ok {
		ch.SyncVarOnChange[id] = fnc
	}
}
func (ch *ClientHandler) UpdateSyncVars() (uc int) {
	var_data := []byte{SYNCVAR_UPDATE}
	for id,sv := range(ch.SyncvarsByID) {
		if sv.IsDirty() {
			uc ++
			syncDat := sv.GetData()
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			payload := append(cmp.Int16ToBytes(int16(id)), data...)
			var_data = append(var_data, payload...)
			printLogF("#####Server: Updating SyncVar with ID=%v: len(dat)=%v, initiated by self=%s", id, len(syncDat), ch.Conn.LocalAddr().String())
		}
	}
	if len(var_data) > 1 {
		ch.Server.Send(var_data, ch.Server.ConnToIdx[ch.Conn])
	}
	return 
}
func (ch *ClientHandler) DeleteSyncVar(ACID int) {
	id := ch.ACIDToID[ACID]
	printLogF("#####Server: Deleting SyncVar with ID=%v, ACID='%v', initiated by self=%s", id, ACID, ch.Conn.LocalAddr().String())
	ch.deleteSyncVarLocal(ACID, id)
	ch.deleteSyncVarRemote(ACID, id)
}
func (ch *ClientHandler) deleteSyncVarLocal(ACID int, id int) {
	delete(ch.SyncvarsByID, id)
	delete(ch.SyncvarsByACID, ACID)
	delete(ch.ACIDToID, ACID)
}
func (ch *ClientHandler) deleteSyncVarRemote(ACID int, id int) {
	data := append(cmp.Int16ToBytes(int16(id)), cmp.Int16ToBytes(int16(ACID))...)
	ch.Server.Send(append([]byte{SYNCVAR_DELETION}, data...), ch.Server.ConnToIdx[ch.Conn])
}
func (ch *ClientHandler) onSyncVarUpdateC(data []byte) {
	for true {
		id := int(cmp.BytesToInt16(data[:2]))
		l := cmp.BytesToInt16(data[2:4])
		dat := data[4:4+l]
		data = data[4+l:]
		ch.SyncvarsByID[id].SetData(dat)
		if fnc, ok := ch.SyncVarOnChange[id]; ok {
			defer fnc(ch.SyncvarsByID[id], id)
		}
		if len(data) <= 0 {
			break
		}
		printLogF("#####Server: Updating SyncVar with ID=%v: len(dat)=%v, initiated by client=%s", id, l, ch.Conn.RemoteAddr().String())
	}
}
func (ch *ClientHandler) processSyncVarRegistry(data []byte) {
	t := data[0]
	ACID := int(cmp.BytesToInt16(data[1:3]))
	id := ch.idCounter
	ch.SyncvarsByID[id] = GetSyncVarOfType(t)
	ch.SyncvarsByACID[ACID] = ch.SyncvarsByID[id]
	ch.ACIDToID[ACID] = id
	resp := append(append([]byte{SYNCVAR_REGISTRY_CONFIRMATION}, data[1:3]...), cmp.Int16ToBytes(int16(id))...)
	ch.Server.Send(resp, ch.Server.ConnToIdx[ch.Conn])
	ch.idCounter ++
	printLogF("#####Server: Creating SyncVar with ID=%v, ACID='%v' , initiated by client=%s", id, ACID, ch.Conn.RemoteAddr().String())
}

func GetServerManager(s *Server) (sm *ServerManager) {
	sm = &ServerManager{Server:s, Handler:make(map[*ws.Conn]*ClientHandler), AllClients:make([]*ws.Conn, 0)}
	s.InputHandler = 	sm.receive
	s.OnNewConn = 		sm.onNewConn
	s.OnCloseConn = 	sm.onCloseConn
	sm.StandardOnChange = nil
	return
}
type ServerManager struct {
	Server   *Server
	Handler map[*ws.Conn]*ClientHandler
	AllClients []*ws.Conn
	StandardOnChange func(SyncVar, int)
	
	InputHandler func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnNewConn    func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
	OnCloseConn  func(c *ws.Conn, mt int, msg []byte, err error, s *Server)
}
func (m *ServerManager) RegisterSyncVarToAllClients(sv SyncVar, ACID int) {
	m.RegisterSyncVar(sv, ACID, m.AllClients...)
}
func (m *ServerManager) RegisterSyncVarsToAllClients(svs []SyncVar, ACIDs []int) {
	m.RegisterSyncVars(svs, ACIDs, m.AllClients...)
}
func (m *ServerManager) RegisterSyncVars(svs []SyncVar, ACIDs []int, clients ...*ws.Conn) {
	for _,sv := range(svs) {
		sv.IsRegisteredTo(len(clients))
	}
	for _,c := range(clients) {
		m.Handler[c].RegisterSyncVars(svs, ACIDs...)
		m.Server.WaitForConfirmation(m.Server.ConnToIdx[c])
		if m.StandardOnChange != nil {
			for _,ACID := range(ACIDs) {
				m.Handler[c].RegisterOnChangeFunc(ACID, m.StandardOnChange)
			}
		}
	}
}
func (m *ServerManager) RegisterSyncVar(sv SyncVar, ACID int, clients ...*ws.Conn) {
	sv.IsRegisteredTo(len(clients))
	for _,c := range(clients) {
		m.Handler[c].RegisterSyncVar(sv, ACID)
		m.Server.WaitForConfirmation(m.Server.ConnToIdx[c])
		if m.StandardOnChange != nil {
			m.Handler[c].RegisterOnChangeFunc(ACID, m.StandardOnChange)
		}
	}
}
func (m *ServerManager) RegisterOnChangeFunc(ACID int, fncs []func(SyncVar, int), clients ...*ws.Conn) {
	fnc := fncs[0]
	for i,c := range(clients) {
		if i > 0 && i < len(fncs) {
			fnc = fncs[i]
		}
		m.Handler[c].RegisterOnChangeFunc(ACID, fnc)
	}
}
func (m *ServerManager) RegisterOnChangeFuncToAllClients(ACID int, fnc func(SyncVar, int)) {
	for _,c := range(m.AllClients) {
		m.Handler[c].RegisterOnChangeFunc(ACID, fnc)
	}
}
func (m *ServerManager) UpdateSyncVars() {
	for _,h := range(m.Handler) {
		h.UpdateSyncVars()
	}
}
func (m *ServerManager) DeleteSyncVar(ACID int, clients ...*ws.Conn) {
	for _,c := range(clients) {
		m.Handler[c].DeleteSyncVar(ACID)
	}
}
func (m *ServerManager) GetHandler(id int) *ClientHandler {
	if id >= 0 && id < len(m.AllClients) {
		return m.Handler[m.AllClients[id]]
	}
	return nil
}

func (m *ServerManager) receive(c *ws.Conn, mt int, msg []byte, err error, s *Server) {
	if err != nil {
		m.onCloseConn(c,mt,msg,err,s)
		return
	}
	if msg[0] == SYNCVAR_REGISTRY {
		m.Handler[c].processSyncVarRegistry(msg[1:])
	}else if msg[0] == SYNCVAR_UPDATE {
		m.Handler[c].onSyncVarUpdateC(msg[1:])
	}else if msg[0] == SYNCVAR_DELETION {
		id := int(cmp.BytesToInt16(msg[1:3]))
		ACID := int(cmp.BytesToInt16(msg[3:5]))
		m.Handler[c].deleteSyncVarLocal(ACID, id)
		printLogF("#####Server: Deleting SyncVar with ID=%v, ACID='%v' , initiated by client=%s", id, ACID, c.RemoteAddr().String())
	}
	if m.InputHandler != nil {
		go m.InputHandler(c,mt,msg,err,s)
	}
}
func (m *ServerManager) onNewConn(c *ws.Conn, mt int, msg []byte, err error, s *Server) {
	m.Handler[c] = GetNewClientHandler(m.Server, c)
	m.AllClients = append(m.AllClients, c)
	if m.OnNewConn != nil {
		go m.OnNewConn(c,mt,msg,err,s)
	}
}
func (m *ServerManager) onCloseConn(c *ws.Conn, mt int, msg []byte, err error, s *Server) {
	delete(m.Handler, c)
	m.AllClients = DeleteConn(true, c, m.AllClients...)
	if m.OnCloseConn != nil {
		go m.OnCloseConn(c,mt,msg,err,s)
	}
}