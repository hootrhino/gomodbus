package modbus

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// DeviceRegister and Client are assumed to be defined elsewhere
type OnDataFunc func([]DeviceRegister)
type OnErrorFunc func(error)

// RegisterScheduler handles loading and organizing register groups
type RegisterScheduler struct {
	client     ModbusApi
	groups     [][]DeviceRegister
	clientType string
	mu         sync.Mutex
}

func NewRegisterScheduler(client ModbusApi) *RegisterScheduler {
	return &RegisterScheduler{
		client:     client,
		clientType: client.GetType(),
	}
}

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

func (rs *RegisterScheduler) ReadGrouped() ([][]DeviceRegister, []error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.clientType == "TCP" {
		return ReadGroupedDataConcurrently(rs.client, rs.groups)
	}
	return ReadGroupedDataSequential(rs.client, rs.groups)
}

// RegisterStream handles data pushing and callback dispatch

type RegisterStream struct {
	dataCh  chan []DeviceRegister
	stopCh  chan struct{}
	onData  atomic.Value // holds OnDataFunc
	onError atomic.Value // holds OnErrorFunc
}

func NewRegisterStream(bufferSize int) *RegisterStream {
	rs := &RegisterStream{
		dataCh: make(chan []DeviceRegister, bufferSize),
		stopCh: make(chan struct{}),
	}
	return rs
}

func (rs *RegisterStream) SetOnData(fn OnDataFunc) {
	rs.onData.Store(fn)
}

func (rs *RegisterStream) SetOnError(fn OnErrorFunc) {
	rs.onError.Store(fn)
}

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

func (rs *RegisterStream) Push(data []DeviceRegister) {
	select {
	case rs.dataCh <- data:
	case <-rs.stopCh:
		return
	}
}

func (rs *RegisterStream) Stop() {
	close(rs.stopCh)
}

// ModbusRegisterManager coordinates scheduling and streaming

type ModbusRegisterManager struct {
	Scheduler *RegisterScheduler
	Stream    *RegisterStream
}

func NewModbusRegisterManager(client ModbusApi, bufferSize int) *ModbusRegisterManager {
	return &ModbusRegisterManager{
		Scheduler: NewRegisterScheduler(client),
		Stream:    NewRegisterStream(bufferSize),
	}
}

func (m *ModbusRegisterManager) LoadRegisters(registers []DeviceRegister) error {
	return m.Scheduler.Load(registers)
}

func (m *ModbusRegisterManager) ReadAndStream() []error {
	groups, errs := m.Scheduler.ReadGrouped()
	for _, group := range groups {
		m.Stream.Push(group)
	}
	return errs
}
func (m *ModbusRegisterManager) SetOnData(fn OnDataFunc) {
	m.Stream.SetOnData(fn)
}
func (m *ModbusRegisterManager) SetOnError(fn OnErrorFunc) {
	m.Stream.SetOnError(fn)
}
func (m *ModbusRegisterManager) Start() {
	m.Stream.Start()
}
func (m *ModbusRegisterManager) Stop() {
	m.Stream.Stop()
}

type ModbusDevicePoller struct {
	managers []*ModbusRegisterManager
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewModbusDevicePoller(interval time.Duration) *ModbusDevicePoller {
	return &ModbusDevicePoller{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (dp *ModbusDevicePoller) AddManager(mgr *ModbusRegisterManager) {
	dp.managers = append(dp.managers, mgr)
}

func (dp *ModbusDevicePoller) Start() {
	for _, mgr := range dp.managers {
		mgr.Start()
	}
	dp.wg.Add(1)
	go func() {
		defer dp.wg.Done()
		ticker := time.NewTicker(dp.interval)
		defer ticker.Stop()
		for {
			select {
			case <-dp.stopCh:
				return
			case <-ticker.C:
				var wg sync.WaitGroup
				errCh := make(chan error, len(dp.managers))
				for _, mgr := range dp.managers {
					wg.Add(1)
					go func(m *ModbusRegisterManager) {
						defer wg.Done()
						errs := m.ReadAndStream()
						for _, err := range errs {
							errCh <- err
						}
					}(mgr)
				}
				go func() {
					wg.Wait()
					close(errCh)
				}()
				for err := range errCh {
					for _, mgr := range dp.managers {
						if cb := mgr.Stream.onError.Load(); cb != nil {
							cb.(OnErrorFunc)(err)
						}
					}
				}
			}
		}
	}()
}

func (dp *ModbusDevicePoller) Stop() {
	close(dp.stopCh)
	dp.wg.Wait()
	for _, mgr := range dp.managers {
		mgr.Stop()
	}
}
