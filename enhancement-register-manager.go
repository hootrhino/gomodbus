package modbus

import "sync"

type RegisterManager struct {
	OnReadCallback   func(registers []DeviceRegister)
	OnErrorCallback  func(err error)
	dataQueue        chan []DeviceRegister
	groupedRegisters [][]DeviceRegister
	exitSignal       chan struct{}
	runningChannel   chan struct{}
	client           Client
	clientType       string
	mu               sync.Mutex // Protects shared resources
}

// NewRegisterManager creates a new instance of RegisterManager
func NewRegisterManager(client Client, queueSize int) *RegisterManager {
	return &RegisterManager{
		dataQueue:        make(chan []DeviceRegister, queueSize),
		groupedRegisters: [][]DeviceRegister{},
		exitSignal:       make(chan struct{}),
		runningChannel:   make(chan struct{}),
		client:           client,
		clientType:       client.GetHandlerType(),
	}
}

// SetOnReadCallback sets the callback for successful reads
func (m *RegisterManager) SetOnReadCallback(callback func(registers []DeviceRegister)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OnReadCallback = callback
}

// SetOnErrorCallback sets the callback for errors
func (m *RegisterManager) SetOnErrorCallback(callback func(err error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OnErrorCallback = callback
}

// Start begins processing the data queue
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
				m.mu.Lock()
				if m.OnReadCallback != nil {
					m.OnReadCallback(data)
				}
				m.mu.Unlock()
			}
		}
	}()
}

// LoadRegisters loads and groups the provided registers
func (m *RegisterManager) LoadRegisters(registers []DeviceRegister) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groupedRegisters = m.GroupDeviceRegister(registers)
}

// Stop gracefully stops the manager
func (m *RegisterManager) Stop() {
	close(m.exitSignal)
	<-m.runningChannel
	close(m.dataQueue)
}

// GroupDeviceRegister groups the registers based on address continuity
func (m *RegisterManager) GroupDeviceRegister(registers []DeviceRegister) [][]DeviceRegister {
	return GroupDeviceRegister(registers)
}

// ReadGroupedData reads grouped data either concurrently or sequentially
func (m *RegisterManager) ReadGroupedData() {
	var result [][]DeviceRegister
	if m.clientType == "TCP" {
		result = ReadGroupedDataConcurrently(m.client, m.groupedRegisters)
	} else {
		result = ReadGroupedDataSequential(m.client, m.groupedRegisters)
	}

	for _, group := range result {
		select {
		case m.dataQueue <- group:
		case <-m.exitSignal:
			return
		}
	}
}
