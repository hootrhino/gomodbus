package modbus

import (
	"fmt"
	"sync"
)

type RegisterManager struct {
	OnReadCallback   func(registers []DeviceRegister)
	OnErrorCallback  func(err error)
	dataQueue        chan []DeviceRegister
	groupedRegisters [][]DeviceRegister
	exitSignal       chan struct{}
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
func (m *RegisterManager) LoadRegisters(registers []DeviceRegister) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Check register tag duplication
	tagMap := make(map[string]bool)
	for _, register := range registers {
		if tagMap[register.Tag] {
			m.OnErrorCallback(fmt.Errorf("duplicate tag: %s", register.Tag))
			return fmt.Errorf("duplicate tag: %s", register.Tag)
		}
		tagMap[register.Tag] = true
	}
	m.groupedRegisters = m.GroupDeviceRegister(registers)
	return nil
}

// Stop gracefully stops the manager
func (m *RegisterManager) Stop() {
	close(m.exitSignal)
	close(m.dataQueue)
}

// GroupDeviceRegister groups the registers based on address continuity
func (m *RegisterManager) GroupDeviceRegister(registers []DeviceRegister) [][]DeviceRegister {
	return GroupDeviceRegisterWithUniqueAddress(registers)
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
