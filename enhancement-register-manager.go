package modbus

type RegisterManager struct {
	OnReadCallback   func(registers []DeviceRegister)
	OnErrorCallback  func(err error)
	dataQueue        chan []DeviceRegister
	groupedRegisters [][]DeviceRegister
	exitSignal       chan struct{}
	runningChannel   chan struct{}
	client           Client
	clientType       string
}

func NewRegisterManager(client Client, clientType string, queueSize int) *RegisterManager {
	return &RegisterManager{
		dataQueue:        make(chan []DeviceRegister, queueSize),
		groupedRegisters: [][]DeviceRegister{},
		exitSignal:       make(chan struct{}),
		runningChannel:   make(chan struct{}),
		client:           client,
		clientType:       clientType,
	}
}
func (m *RegisterManager) SetOnReadCallback(callback func(registers []DeviceRegister)) {
	m.OnReadCallback = callback
}
func (m *RegisterManager) SetOnErrorCallback(callback func(err error)) {
	m.OnErrorCallback = callback
}
func (m *RegisterManager) Start() {
	go func() {
		defer close(m.runningChannel)
		for {
			select {
			case <-m.exitSignal:
				return
			case data, ok := <-m.dataQueue:
				if !ok {
					return
				}
				m.OnReadCallback(data)
			}
		}
	}()
}
func (m *RegisterManager) LoadRegisters(registers []DeviceRegister) {
	m.groupedRegisters = m.GroupDeviceRegister(registers)
}
func (m *RegisterManager) Stop() {
	close(m.exitSignal)
	<-m.runningChannel
	close(m.dataQueue)
}

func (m *RegisterManager) GroupDeviceRegister(registers []DeviceRegister) [][]DeviceRegister {
	return GroupDeviceRegister(registers)
}

func (m *RegisterManager) ReadGroupedData() {
	if m.clientType == "TCP" {
		result := ReadGroupedDataConcurrently(m.client, m.groupedRegisters)
		for _, group := range result {
			m.dataQueue <- group
		}
	} else {
		result := ReadGroupedDataSequential(m.client, m.groupedRegisters)
		for _, group := range result {
			m.dataQueue <- group
		}
	}

}
