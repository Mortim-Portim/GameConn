package GC

type ClientManager struct {
	Client   *Client
	Syncvars []SyncVar
}

func (m *ClientManager) RegisterVar(sv SyncVar) {
	m.Syncvars = append(m.Syncvars, sv)
}

func (m *ClientManager) Update() {
	for _, sv := range m.Syncvars {
		if sv.isDirty() {

		}
	}
}

func (m *ClientManager) Receive(input []byte) {
	m.Syncvars[int(input[0])].setData(input[1:])
}

type ServerManager struct {
	Server   *Server
	syncvars []SyncVar
}

func (m *ServerManager) RegisterVar(sv SyncVar) {
	m.syncvars = append(m.syncvars, sv)
}

func (m *ServerManager) Update() {
	for i, sv := range m.syncvars {
		if sv.isDirty() {
			adress := []byte{byte(i)}
			content := sv.getData()

			message := append(adress, content...)
			m.Server.SendAll(message)
		}
	}
}
