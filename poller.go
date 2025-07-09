// Copyright (C) 2024  wwhai
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <https://www.gnu.org/licenses/>.

package modbus

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// DeviceRegister and Client are assumed to be defined elsewhere

// OnDataFunc is a callback type for pushing register data
type OnDataFunc func([]DeviceRegister)

// OnErrorFunc is a callback type for error reporting
type OnErrorFunc func(error)

// RegisterScheduler handles grouping and scheduling of Modbus register reads
type RegisterScheduler struct {
	client     ModbusApi          // Modbus client instance
	groups     [][]DeviceRegister // Grouped registers for batch reading
	clientType string             // "TCP" or other, affects concurrency
	mu         sync.Mutex         // Protects groups and clientType
}

// NewRegisterScheduler creates a new RegisterScheduler for a Modbus client
func NewRegisterScheduler(client ModbusApi) *RegisterScheduler {
	return &RegisterScheduler{
		client:     client,
		clientType: client.GetMode(),
	}
}

// Load validates and groups registers for efficient polling
func (rs *RegisterScheduler) Load(registers []DeviceRegister) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	tagMap := make(map[string]bool)
	for _, r := range registers {
		if tagMap[r.Tag] {
			return fmt.Errorf("duplicate tag: %s", r.Tag)
		}
		tagMap[r.Tag] = true
	}
	rs.groups = GroupDeviceRegisterWithLogicalContinuity(registers)
	return nil
}

// ReadGrouped reads all register groups, using concurrency for TCP clients
func (rs *RegisterScheduler) ReadGrouped() ([][]DeviceRegister, []error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.clientType == "TCP" {
		return ReadGroupedDataConcurrently(rs.client, rs.groups)
	}
	return ReadGroupedDataSequential(rs.client, rs.groups)
}

// RegisterStream handles asynchronous data pushing and callback dispatch
type RegisterStream struct {
	dataCh  chan []DeviceRegister // Channel for pushing register data
	stopCh  chan struct{}         // Channel to signal stop
	onData  atomic.Value          // Stores OnDataFunc callback
	onError atomic.Value          // Stores OnErrorFunc callback
}

// NewRegisterStream creates a RegisterStream with a given buffer size
func NewRegisterStream(bufferSize int) *RegisterStream {
	rs := &RegisterStream{
		dataCh: make(chan []DeviceRegister, bufferSize),
		stopCh: make(chan struct{}),
	}
	return rs
}

// SetOnData sets the callback for data events
func (rs *RegisterStream) SetOnData(fn OnDataFunc) {
	rs.onData.Store(fn)
}

// SetOnError sets the callback for error events
func (rs *RegisterStream) SetOnError(fn OnErrorFunc) {
	rs.onError.Store(fn)
}

// Start launches the goroutine to dispatch data to the OnData callback
func (rs *RegisterStream) Start() {
	go func() {
		for {
			select {
			case <-rs.stopCh:
				return
			case data, ok := <-rs.dataCh:
				if !ok {
					return
				}
				if cb := rs.onData.Load(); cb != nil {
					cb.(OnDataFunc)(data)
				}
			}
		}
	}()
}

// Push sends register data to the stream, unless stopped
func (rs *RegisterStream) Push(data []DeviceRegister) {
	select {
	case rs.dataCh <- data:
	case <-rs.stopCh:
		return
	}
}

// Stop signals the stream to stop processing
func (rs *RegisterStream) Stop() {
	close(rs.stopCh)
}

// ModbusRegisterManager coordinates register scheduling and streaming
type ModbusRegisterManager struct {
	Scheduler *RegisterScheduler // Handles grouping and scheduling
	Stream    *RegisterStream    // Handles data streaming and callbacks
}

// NewModbusRegisterManager creates a new manager for a Modbus client
func NewModbusRegisterManager(client ModbusApi, bufferSize int) *ModbusRegisterManager {
	return &ModbusRegisterManager{
		Scheduler: NewRegisterScheduler(client),
		Stream:    NewRegisterStream(bufferSize),
	}
}

// LoadRegisters loads and groups registers for polling
func (m *ModbusRegisterManager) LoadRegisters(registers []DeviceRegister) error {
	return m.Scheduler.Load(registers)
}

// ReadAndStream reads all register groups and pushes them to the stream
func (m *ModbusRegisterManager) ReadAndStream() []error {
	groups, errs := m.Scheduler.ReadGrouped()
	for _, group := range groups {
		m.Stream.Push(group)
	}
	return errs
}

// SetOnData sets the data callback for the stream
func (m *ModbusRegisterManager) SetOnData(fn OnDataFunc) {
	m.Stream.SetOnData(fn)
}

// SetOnError sets the error callback for the stream
func (m *ModbusRegisterManager) SetOnError(fn OnErrorFunc) {
	m.Stream.SetOnError(fn)
}

// Start launches the stream's goroutine
func (m *ModbusRegisterManager) Start() {
	m.Stream.Start()
}

// Stop signals the stream to stop
func (m *ModbusRegisterManager) Stop() {
	m.Stream.Stop()
}

// ModbusDevicePoller is responsible for polling Modbus registers at a specified interval.
type ModbusDevicePoller struct {
	managers []*ModbusRegisterManager
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewModbusDevicePoller creates a new ModbusDevicePoller with the given interval.
func NewModbusDevicePoller(interval time.Duration) *ModbusDevicePoller {
	return &ModbusDevicePoller{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// AddManager adds a ModbusRegisterManager to the poller.
func (dp *ModbusDevicePoller) AddManager(mgr *ModbusRegisterManager) {
	dp.managers = append(dp.managers, mgr)
}

// Start initiates the polling process.
func (dp *ModbusDevicePoller) Start() {
	for _, mgr := range dp.managers {
		mgr.Start()
	}
	dp.wg.Add(1)
	go dp.poll()
}

// poll is a private method that runs the polling loop.
func (dp *ModbusDevicePoller) poll() {
	defer dp.wg.Done()
	ticker := time.NewTicker(dp.interval)
	defer ticker.Stop()

	for {
		select {
		case <-dp.stopCh:
			return
		case <-ticker.C:
			dp.pollManagers()
		}
	}
}

// pollManagers reads and streams data from all registered managers.
func (dp *ModbusDevicePoller) pollManagers() {
	var wg sync.WaitGroup
	for _, mgr := range dp.managers {
		wg.Add(1)
		go func(m *ModbusRegisterManager) {
			defer wg.Done()
			m.ReadAndStream()
		}(mgr)
	}
	wg.Wait()
}

// Stop stops the polling process and cleans up resources.
func (dp *ModbusDevicePoller) Stop() {
	close(dp.stopCh)
	dp.wg.Wait()
	for _, mgr := range dp.managers {
		mgr.Stop()
	}
}
