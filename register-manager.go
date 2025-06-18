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
	client           ModbusApi
	clientType       string
	closed           bool
	mu               sync.Mutex // Protects shared resources
}

// NewRegisterManager creates a new instance of RegisterManager
func NewRegisterManager(client ModbusApi, queueSize int) *RegisterManager {
	return &RegisterManager{
		dataQueue:        make(chan []DeviceRegister, queueSize),
		groupedRegisters: [][]DeviceRegister{},
		exitSignal:       make(chan struct{}),
		client:           client,
		clientType:       client.GetMode(),
		closed:           false,
	}
}

// GetClient returns the Modbus client instance
func (m *RegisterManager) GetClient() ModbusApi {
	return m.client
}

// GetClientType returns the type of the client
func (m *RegisterManager) GetClientType() string {
	return m.clientType
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
	// Filter out virtual registers
	tempRegisters := make([]DeviceRegister, 0, len(registers))
	for _, register := range registers {
		if register.Type != "virtual" {
			tempRegisters = append(tempRegisters, register)
		}
	}
	m.groupedRegisters = m.GroupDeviceRegister(tempRegisters)
	return nil
}

// Stop gracefully stops the manager
func (m *RegisterManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.closed = true
	close(m.exitSignal)
	close(m.dataQueue)
}

// GroupDeviceRegister groups the registers based on address continuity
func (m *RegisterManager) GroupDeviceRegister(registers []DeviceRegister) [][]DeviceRegister {
	return GroupDeviceRegisterWithLogicalContinuity(registers)
}

// ReadGroupedData reads grouped data either concurrently or sequentially
func (m *RegisterManager) ReadGroupedData() []error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return []error{fmt.Errorf("register manager is closed")}
	}
	var result [][]DeviceRegister
	var errors []error
	if m.clientType == "TCP" {
		result, errors = ReadGroupedDataConcurrently(m.client, m.groupedRegisters)
	} else {
		result, errors = ReadGroupedDataSequential(m.client, m.groupedRegisters)
	}

	for _, group := range result {
		select {
		case m.dataQueue <- group:
		case _, ok := <-m.exitSignal:
			if !ok {
				return nil
			}
			return nil
		}
	}
	return errors
}
