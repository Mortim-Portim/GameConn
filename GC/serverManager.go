package GC

import (
	"sync"
	ws "github.com/gorilla/websocket"
	cmp "github.com/mortim-portim/GraphEng/Compression"
)

type Handler interface {
	RegisterSyncVar(sv SyncVar, ACID int)
	RegisterSyncVars(svs map[int]SyncVar)
	RegisterOnChangeFunc(ACID int, fnc func(SyncVar, int))
	UpdateSyncVarsWithACIDs(...int) int
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
	SV_ID_L, SV_ACID_L, ACID_ID_L, SV_chng sync.Mutex
	idCounter int
}
func (ch *ClientHandler) RegisterSyncVar(sv SyncVar, ACID int) {
	printLogF("#####Server: Creating SyncVar with ID=%v, ACID='%v', type=%v , on conn %p\n", ch.idCounter, ACID, sv.Type(), ch.Conn)
	ch.SV_ACID_L.Lock()
	ch.SyncvarsByACID[ACID] 		= sv
	ch.SV_ACID_L.Unlock()
	ch.SV_ID_L.Lock()
	ch.SyncvarsByID[ch.idCounter] 	= sv
	ch.SV_ID_L.Unlock()
	ch.ACID_ID_L.Lock()
	ch.ACIDToID[ACID] 				= ch.idCounter
	ch.ACID_ID_L.Unlock()
	resp := append(append([]byte{SYNCVAR_REGISTRY, sv.Type()}, cmp.Int16ToBytes(int16(ACID))...), cmp.Int16ToBytes(int16(ch.idCounter))...)
	ch.Server.Send(resp, ch.Conn)
	ch.idCounter ++
}
func (ch *ClientHandler) RegisterSyncVars(svs map[int]SyncVar) {
	data := []byte{SYNCVAR_M_REGISTRY}
	for ACID,sv := range(svs) {
		//printLogF("#####Server: MCreating SyncVar with ID=%v, ACID='%v', type=%v , initiated by self=%s", ch.idCounter, ACID, sv.Type(), ch.Conn.LocalAddr().String())
		ch.SV_ACID_L.Lock()
		ch.SyncvarsByACID[ACID] 		= sv
		ch.SV_ACID_L.Unlock()
		ch.SV_ID_L.Lock()
		ch.SyncvarsByID[ch.idCounter] 	= sv
		ch.SV_ID_L.Unlock()
		ch.ACID_ID_L.Lock()
		ch.ACIDToID[ACID] 				= ch.idCounter
		ch.ACID_ID_L.Unlock()
		data = append(data, sv.Type())
		data = append(data, cmp.Int16ToBytes(int16(ACID))...)
		data = append(data, cmp.Int16ToBytes(int16(ch.idCounter))...)
		ch.idCounter ++
	}
	printLogF("#####Server: MCreating %v SyncVars, on conn %p\n", len(svs), ch.Conn)
	ch.Server.Send(data, ch.Conn)
}
func (ch *ClientHandler) RegisterOnChangeFunc(ACID int, fnc func(SyncVar, int)) {
	ch.ACID_ID_L.Lock()
	id, ok := ch.ACIDToID[ACID]
	ch.ACID_ID_L.Unlock()
	if ok {
		ch.SV_chng.Lock()
		ch.SyncVarOnChange[id] = fnc
		ch.SV_chng.Unlock()
	}
}
func (ch *ClientHandler) UpdateSyncVarsWithACIDs(ACIDs ...int) (uc int) {
	var_data := []byte{SYNCVAR_UPDATE}
	ch.SV_ACID_L.Lock()
	for ACID,sv := range(ch.SyncvarsByACID) {
		if sv.IsDirty() && containsI(ACIDs, ACID) {
			uc ++
			syncDat := sv.GetData()
			//printLogF(".....Client: Updating SyncVar with ID=%v: len(dat)=%v, initiated by self=%s", id, len(syncDat), m.Client.LocalAddr().String())
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			ch.ACID_ID_L.Lock()
			id := int16(ch.ACIDToID[ACID])
			ch.ACID_ID_L.Unlock()
			payload := append(cmp.Int16ToBytes(id), data...)
			var_data = append(var_data, payload...)
		}
	}
	ch.SV_ACID_L.Unlock()
	if uc > 0 {
		printLogF("#####Server: Updating %v SyncVars, on conn %p\n", uc, ch.Conn)
	}
	if len(var_data) > 1 {
		ch.Server.Send(var_data, ch.Conn)
	}
	return
}
func (ch *ClientHandler) UpdateSyncVars() (uc int) {
	var_data := []byte{SYNCVAR_UPDATE}
	ch.SV_ID_L.Lock()
	for id,sv := range(ch.SyncvarsByID) {
		if sv.IsDirty() {
			uc ++
			syncDat := sv.GetData()
			data := append(cmp.Int16ToBytes(int16(len(syncDat))), syncDat...)
			payload := append(cmp.Int16ToBytes(int16(id)), data...)
			var_data = append(var_data, payload...)
			//printLogF("#####Server: Updating SyncVar with ID=%v: len(dat)=%v, initiated by self=%s", id, len(syncDat), ch.Conn.LocalAddr().String())
		}
	}
	ch.SV_ID_L.Unlock()
	if uc > 0 {
		printLogF("#####Server: Updating %v SyncVars, on conn %p\n", uc, ch.Conn)
	}
	if len(var_data) > 1 {
		ch.Server.Send(var_data, ch.Conn)
	}
	return 
}
func (ch *ClientHandler) DeleteSyncVar(ACID int) {
	ch.ACID_ID_L.Lock()
	id := ch.ACIDToID[ACID]
	ch.ACID_ID_L.Unlock()
	printLogF("#####Server: Deleting SyncVar with ID=%v, ACID='%v', on conn %p\n", id, ACID, ch.Conn)
	ch.deleteSyncVarLocal(ACID, id)
	ch.deleteSyncVarRemote(ACID, id)
}
func (ch *ClientHandler) deleteSyncVarLocal(ACID int, id int) {
	ch.SV_ID_L.Lock()
	delete(ch.SyncvarsByID, id)
	ch.SV_ID_L.Unlock()
	ch.SV_ACID_L.Lock()
	delete(ch.SyncvarsByACID, ACID)
	ch.SV_ACID_L.Unlock()
	ch.ACID_ID_L.Lock()
	delete(ch.ACIDToID, ACID)
	ch.ACID_ID_L.Unlock()
}
func (ch *ClientHandler) deleteSyncVarRemote(ACID int, id int) {
	data := append(cmp.Int16ToBytes(int16(id)), cmp.Int16ToBytes(int16(ACID))...)
	ch.Server.Send(append([]byte{SYNCVAR_DELETION}, data...), ch.Conn)
}
func (ch *ClientHandler) onSyncVarUpdateC(data []byte) {
	num := 0
	for true {
		num ++
		id := int(cmp.BytesToInt16(data[:2]))
		l := cmp.BytesToInt16(data[2:4])
		dat := data[4:4+l]
		data = data[4+l:]
		ch.SV_ID_L.Lock()
		sv := ch.SyncvarsByID[id]
		ch.SV_ID_L.Unlock()
		sv.SetData(dat)
		ch.SV_chng.Lock()
		if fnc, ok := ch.SyncVarOnChange[id]; ok {
			defer fnc(sv, id)
		}
		ch.SV_chng.Unlock()
		if len(data) <= 0 {
			break
		}
		//printLogF("#####Server: Updating SyncVar with ID=%v: len(dat)=%v, initiated by client=%s", id, l, ch.Conn.RemoteAddr().String())
	}
	printLogF("#####Server: Updating %v SyncVars, from conn %p\n", num, ch.Conn)
}
func (ch *ClientHandler) processSyncVarRegistry(data []byte) {
	t := data[0]
	ACID := int(cmp.BytesToInt16(data[1:3]))
	id := ch.idCounter
	ch.SV_ID_L.Lock()
	ch.SyncvarsByID[id] = GetSyncVarOfType(t)
	ch.SV_ACID_L.Lock()
	ch.SyncvarsByACID[ACID] = ch.SyncvarsByID[id]
	ch.SV_ACID_L.Unlock()
	ch.SV_ID_L.Unlock()
	ch.ACID_ID_L.Lock()
	ch.ACIDToID[ACID] = id
	ch.ACID_ID_L.Unlock()
	resp := append(append([]byte{SYNCVAR_REGISTRY_CONFIRMATION}, data[1:3]...), cmp.Int16ToBytes(int16(id))...)
	ch.Server.Send(resp, ch.Conn)
	ch.idCounter ++
	printLogF("#####Server: Creating SyncVar with ID=%v, ACID='%v', from conn %p\n", id, ACID, ch.Conn)
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
func (m *ServerManager) RegisterSyncVarsToAllClients(svs map[int]SyncVar) {
	m.RegisterSyncVars(svs, m.AllClients...)
}
func (m *ServerManager) RegisterSyncVars(svs map[int]SyncVar, clients ...*ws.Conn) {
	for _,sv := range(svs) {
		sv.IsRegisteredTo(len(clients))
	}
	for _,c := range(clients) {
		m.Handler[c].RegisterSyncVars(svs)
		m.Server.WaitForConfirmation(c)
		if m.StandardOnChange != nil {
			for ACID,_ := range(svs) {
				m.Handler[c].RegisterOnChangeFunc(ACID, m.StandardOnChange)
			}
		}
	}
}
func (m *ServerManager) RegisterSyncVar(sv SyncVar, ACID int, clients ...*ws.Conn) {
	sv.IsRegisteredTo(len(clients))
	for _,c := range(clients) {
		m.Handler[c].RegisterSyncVar(sv, ACID)
		m.Server.WaitForConfirmation(c)
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
		printLogF("#####Server: Deleting SyncVar with ID=%v, ACID='%v', from conn %p\n", id, ACID, c)
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