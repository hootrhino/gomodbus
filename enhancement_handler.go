package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

// DefaultLogger is a simple logger that implements the io.Writer interface.
type DefaultLogger struct {
}

// Write implements the io.Writer interface for DefaultLogger.
func (l *DefaultLogger) Write(p []byte) (n int, err error) {
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("[%s] %s\n", timeStr, string(p))
	return len(p), nil
}

// Standard Response PDU Lengths (Including Function Code, Excluding Slave ID and CRC)
const (
	RespPDULenWriteSingleCoil        = 1 + 2 + 2 // FuncCode (1) + Address (2) + Value (2)
	RespPDULenWriteSingleRegister    = 1 + 2 + 2 // FuncCode (1) + Address (2) + Value (2)
	RespPDULenWriteMultipleCoils     = 1 + 2 + 2 // FuncCode (1) + Address (2) + Quantity (2)
	RespPDULenWriteMultipleRegisters = 1 + 2 + 2 // FuncCode (1) + Address (2) + Quantity (2)
	RespPDULenReadExceptionStatus    = 1 + 1     // FuncCode (1) + Status Byte (1)
	// RespPDULenReadDeviceIdentity is dynamic
)

// ModbusHandler implements the ModbusApi interface for handling Modbus requests.
type ModbusHandler struct {
	logger         io.Writer       // Logger for debug output
	rtuTransporter *RTUTransporter // New field for RTU transporter
	tcpTransporter *TCPTransporter // New field for TCP transporter
	transmissionID uint16          // Track the current transaction ID
	mode           string          // "rtu" or "tcp"
}

// GetType implements ModbusApi.
func (h *ModbusHandler) GetType() string {
	return h.mode
}

// NewModbusHandler creates a new ModbusHandler with the given serial port and timeout.
// It returns an instance implementing the ModbusApi interface.
func NewModbusRTUHandler(port io.ReadWriteCloser, timeout time.Duration) ModbusApi {
	return &ModbusHandler{
		logger:         &DefaultLogger{},
		mode:           "rtu",
		rtuTransporter: NewRTUTransporter(port, timeout),
	}
}
func NewModbusTCPHandler(conn net.Conn, timeout time.Duration) ModbusApi {
	return &ModbusHandler{
		logger:         &DefaultLogger{},
		mode:           "tcp",
		tcpTransporter: NewTCPTransporter(conn, timeout, nil),
	}
}

func (h *ModbusHandler) SetLogger(logger io.Writer) {
	h.logger = logger
}

// readModbusData sends a standard read request (address + quantity PDU data)
// and performs basic response validation (function code, byte count length check).
// It returns the data payload from the response PDU (after function code and byte count).
// This helper is used by ReadCoils, ReadDiscreteInputs, ReadHoldingRegisters, ReadInputRegisters.
func (h *ModbusHandler) readModbusData(funcCode uint8, slaveID uint16, startAddress, quantity uint16) ([]byte, error) {
	// Build PDU data part (address + quantity)
	pduData := make([]byte, 4)
	binary.BigEndian.PutUint16(pduData[0:2], startAddress)
	binary.BigEndian.PutUint16(pduData[2:4], quantity)

	// Build the full request PDU (func code + PDU data)
	// Assumes buildRequestPDU prepends the funcCode to pduData
	reqPDU, err := buildRequestPDU(funcCode, pduData) // Assumes buildRequestPDU exists
	if err != nil {
		// Log the error if logger is available
		if h.logger != nil {
			fmt.Fprintf(h.logger, "modbus: Error building request PDU for func %02X (slave %d): %v", funcCode, slaveID, err)
		}
		return nil, fmt.Errorf("modbus: failed to build request PDU for func %02X (slave %d): %w", funcCode, slaveID, err)
	}

	// Send request and receive response
	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		// sendAndReceive already handles transport errors and Modbus exceptions
		return nil, fmt.Errorf("modbus: send/receive failed for func %02X (slave %d): %w", funcCode, slaveID, err)
	}

	// Validate response function code
	// sendAndReceive checks for exception responses (funcCode | 0x80),
	// so here we just check if the received func code matches the requested one.
	if len(respPDU) == 0 || respPDU[0] != funcCode {
		// This should ideally not happen if sendAndReceive worked and returned a non-exception
		return nil, fmt.Errorf("modbus: unexpected function code in response for func %02X (slave %d): got %d", funcCode, slaveID, respPDU[0])
	}

	// For standard read function codes (0x01, 0x02, 0x03, 0x04),
	// the response PDU structure is typically:
	// [0] Function Code (1 byte)
	// [1] Byte Count (1 byte) - Number of subsequent data bytes
	// [2...] Data Payload (Byte Count bytes)

	if len(respPDU) < 2 {
		// Response is too short, should at least contain func code and byte count
		return nil, fmt.Errorf("modbus: invalid response length for func %02X (slave %d): expected at least 2 bytes, got %d", funcCode, slaveID, len(respPDU))
	}

	byteCount := int(respPDU[1])
	// Validate the length of the data payload based on byte count
	if len(respPDU) != 2+byteCount {
		return nil, fmt.Errorf("modbus: invalid response data length for func %02X (slave %d): expected %d bytes, got %d", funcCode, slaveID, byteCount, len(respPDU)-2)
	}

	// Return the data payload part of the response PDU
	return respPDU[2 : 2+byteCount], nil
}

// writeModbusData sends a standard write request, performs basic response validation,
// and returns the full response PDU.
// This helper is used by WriteSingleCoil, WriteSingleRegister, WriteMultipleCoils, WriteMultipleRegisters.
// expectedRespPDULen is the expected length of the response PDU (including func code).
func (h *ModbusHandler) writeModbusData(funcCode uint8, slaveID uint16, pduData []byte, expectedRespPDULen int) ([]byte, error) {
	// Build the full request PDU (func code + PDU data)
	reqPDU, err := buildRequestPDU(funcCode, pduData) // Assumes buildRequestPDU exists
	if err != nil {
		if h.logger != nil {
			fmt.Fprintf(h.logger, "modbus: Error building request PDU for func %02X (slave %d): %v", funcCode, slaveID, err)
		}
		return nil, fmt.Errorf("modbus: failed to build request PDU for func %02X (slave %d): %w", funcCode, slaveID, err)
	}

	// Send request and receive response
	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		// sendAndReceive already handles transport errors and Modbus exceptions
		return nil, fmt.Errorf("modbus: send/receive failed for func %02X (slave %d): %w", funcCode, slaveID, err)
	}

	// Validate response function code
	if len(respPDU) == 0 || respPDU[0] != funcCode {
		return nil, fmt.Errorf("modbus: unexpected function code in response for func %02X (slave %d): got %d", funcCode, slaveID, respPDU[0])
	}

	// Validate response length
	if len(respPDU) != expectedRespPDULen {
		return nil, fmt.Errorf("modbus: invalid response length for func %02X (slave %d): expected %d bytes, got %d", funcCode, slaveID, expectedRespPDULen, len(respPDU))
	}

	// Return the full response PDU for caller to validate echoed data
	return respPDU, nil
}

// ReadCoils reads the specified number of coils starting from the given address.
func (h *ModbusHandler) ReadCoils(slaveID uint16, startAddress, quantity uint16) ([]bool, error) {
	// Use generic read helper to get data payload
	data, err := h.readModbusData(FuncCodeReadCoils, slaveID, startAddress, quantity)
	if err != nil {
		return nil, err // Error is already wrapped by readModbusData
	}

	// Parse coil states from the data payload
	coils := make([]bool, quantity)
	byteCount := len(data) // data is the payload after func code and byte count

	for i := 0; i < int(quantity); i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		// The byteIndex < byteCount check is implicitly handled by readModbusData's length check,
		// but kept here for clarity or if quantity > actual received bits (should ideally not happen)
		if byteIndex < byteCount {
			if (data[byteIndex] & (1 << bitIndex)) != 0 {
				coils[i] = true
			}
		} else {
			// This case indicates more quantity requested than bits received.
			// Depending on strictness, could return error or just stop parsing.
			// Stopping is safer if quantity is somehow inconsistent.
			// if h.logger != nil { fmt.Fprintf(h.logger, "modbus: Warning: Quantity %d exceeds received data bits for coils (slave %d, address %d)", quantity, slaveID, startAddress) }
			break // Stop processing if we run out of data
		}
	}

	return coils, nil
}

// ReadDiscreteInputs reads the specified number of discrete inputs starting from the given address.
func (h *ModbusHandler) ReadDiscreteInputs(slaveID uint16, startAddress, quantity uint16) ([]bool, error) {
	// Use generic read helper to get data payload
	data, err := h.readModbusData(FuncCodeReadDiscreteInputs, slaveID, startAddress, quantity)
	if err != nil {
		return nil, err // Error is already wrapped by readModbusData
	}

	// Parse discrete input states from the data payload
	inputs := make([]bool, quantity)
	byteCount := len(data) // data is the payload after func code and byte count

	for i := 0; i < int(quantity); i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if byteIndex < byteCount {
			if (data[byteIndex] & (1 << bitIndex)) != 0 {
				inputs[i] = true
			}
		} else {
			// if h.logger != nil { fmt.Fprintf(h.logger, "modbus: Warning: Quantity %d exceeds received data bits for discrete inputs (slave %d, address %d)", quantity, slaveID, startAddress) }
			break // Stop processing if we run out of data
		}
	}

	return inputs, nil
}

// ReadHoldingRegisters reads the specified number of holding registers starting from the given address.
func (h *ModbusHandler) ReadHoldingRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error) {
	// Use generic read helper to get data payload
	data, err := h.readModbusData(FuncCodeReadHoldingRegisters, slaveID, startAddress, quantity)
	if err != nil {
		return nil, err // Error is already wrapped by readModbusData
	}

	// Validate the data payload length is even (each register is 2 bytes)
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("modbus: invalid register data length for func %02X (slave %d): expected even number of bytes, got %d", FuncCodeReadHoldingRegisters, slaveID, len(data))
	}

	// Parse register values from the data payload
	registerCount := len(data) / 2
	registers := make([]uint16, registerCount)
	for i := 0; i < registerCount; i++ {
		registers[i] = binary.BigEndian.Uint16(data[2*i : 2*i+2])
	}

	return registers, nil
}

// ReadInputRegisters reads the specified number of input registers starting from the given address.
func (h *ModbusHandler) ReadInputRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error) {
	// Use generic read helper to get data payload
	data, err := h.readModbusData(FuncCodeReadInputRegisters, slaveID, startAddress, quantity)
	if err != nil {
		return nil, err // Error is already wrapped by readModbusData
	}

	// Validate the data payload length is even (each register is 2 bytes)
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("modbus: invalid register data length for func %02X (slave %d): expected even number of bytes, got %d", FuncCodeReadInputRegisters, slaveID, len(data))
	}

	// Parse register values from the data payload
	registerCount := len(data) / 2
	registers := make([]uint16, registerCount)
	for i := 0; i < registerCount; i++ {
		registers[i] = binary.BigEndian.Uint16(data[2*i : 2*i+2])
	}

	return registers, nil
}

// ReadWithMask reads a single holding register and applies an AND/OR mask logically.
// Note: This implementation reads a register (FC 0x03) and performs the mask operation
// in the client code. It does NOT use the Modbus function code 0x16 (Read/Write Multiple Registers),
// which can perform a masked write on the server side.
func (h *ModbusHandler) ReadWithMask(slaveID uint16, readAddress uint16, andMask uint16, orMask uint16) (uint16, error) {
	// Use ReadHoldingRegisters to read the single register
	values, err := h.ReadHoldingRegisters(slaveID, readAddress, 1)
	if err != nil {
		// Error is already wrapped by ReadHoldingRegisters -> readModbusData -> sendAndReceive
		return 0, fmt.Errorf("modbus: failed to read register for ReadWithMask (slave %d, address %d): %w", slaveID, readAddress, err)
	}
	if len(values) == 0 {
		// This case indicates ReadHoldingRegisters returned no values despite no error.
		// Should ideally not happen with quantity 1, but defensively check.
		return 0, fmt.Errorf("modbus: no value returned for ReadWithMask (slave %d, address %d)", slaveID, readAddress)
	}

	// Apply the mask logically in the client
	return uint16(values[0]&andMask | orMask), nil
}

// WriteSingleCoil writes a single coil to the Modbus device.
func (h *ModbusHandler) WriteSingleCoil(slaveID uint16, address uint16, value bool) error {
	// Build PDU data part (address + value)
	pduData := make([]byte, 4)
	binary.BigEndian.PutUint16(pduData[0:2], address)
	if value {
		binary.BigEndian.PutUint16(pduData[2:4], 0xFF00) // ON value
	} else {
		binary.BigEndian.PutUint16(pduData[2:4], 0x0000) // OFF value
	}

	// Use generic write helper
	respPDU, err := h.writeModbusData(FuncCodeWriteSingleCoil, slaveID, pduData, RespPDULenWriteSingleCoil)
	if err != nil {
		return fmt.Errorf("modbus: write single coil failed (slave %d, address %d, value %v): %w", slaveID, address, value, err) // Error already wrapped by writeModbusData
	}

	// Validate echoed data in the response PDU
	respAddress := binary.BigEndian.Uint16(respPDU[1:3])
	respValue := binary.BigEndian.Uint16(respPDU[3:5])

	if respAddress != address {
		return fmt.Errorf("modbus: write single coil response address mismatch (slave %d): expected %d, got %d", slaveID, address, respAddress)
	}

	// Modbus specification dictates echoed value should be 0x0000 or 0xFF00
	if respValue != 0x0000 && respValue != 0xFF00 {
		return fmt.Errorf("modbus: write single coil response value format error (slave %d): expected 0x0000 or 0xFF00, got 0x%04X", slaveID, respValue)
	}

	// Further check if the echoed value matches the requested ON/OFF state
	if value && respValue != 0xFF00 {
		return fmt.Errorf("modbus: write single coil response value mismatch (slave %d): expected ON (0xFF00), got 0x%04X", slaveID, respValue)
	}
	if !value && respValue != 0x0000 {
		return fmt.Errorf("modbus: write single coil response value mismatch (slave %d): expected OFF (0x0000), got 0x%04X", slaveID, respValue)
	}

	return nil
}

// WriteSingleRegister writes a single register to the Modbus device.
func (h *ModbusHandler) WriteSingleRegister(slaveID uint16, address uint16, value uint16) error {
	// Build PDU data part (address + value)
	pduData := make([]byte, 4)
	binary.BigEndian.PutUint16(pduData[0:2], address)
	binary.BigEndian.PutUint16(pduData[2:4], value)

	// Use generic write helper
	respPDU, err := h.writeModbusData(FuncCodeWriteSingleRegister, slaveID, pduData, RespPDULenWriteSingleRegister)
	if err != nil {
		return fmt.Errorf("modbus: write single register failed (slave %d, address %d, value %d): %w", slaveID, address, value, err) // Error already wrapped by writeModbusData
	}

	// Validate echoed data in the response PDU
	respAddress := binary.BigEndian.Uint16(respPDU[1:3])
	respValue := binary.BigEndian.Uint16(respPDU[3:5])

	if respAddress != address {
		return fmt.Errorf("modbus: write single register response address mismatch (slave %d): expected %d, got %d", slaveID, address, respAddress)
	}

	if respValue != value {
		return fmt.Errorf("modbus: write single register response value mismatch (slave %d): expected %d, got %d", slaveID, value, respValue)
	}

	return nil
}

// WriteMultipleCoils writes multiple coils to the Modbus device.
func (h *ModbusHandler) WriteMultipleCoils(slaveID uint16, startAddress uint16, values []bool) error {
	quantity := uint16(len(values))
	byteCount := (quantity + 7) / 8 // Number of bytes needed to hold quantity bits

	// Build PDU data part (startAddress + quantity + byteCount + coilData)
	pduData := make([]byte, 5+byteCount)
	binary.BigEndian.PutUint16(pduData[0:2], startAddress)
	binary.BigEndian.PutUint16(pduData[2:4], quantity)
	pduData[4] = byte(byteCount) // Byte count field

	// Pack boolean values into bytes
	for i := 0; i < int(quantity); i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if values[i] {
			pduData[5+byteIndex] |= (1 << bitIndex) // Set the bit
		}
		// Else: bit is already 0 as slice is zero-initialized
	}

	// Use generic write helper
	respPDU, err := h.writeModbusData(FuncCodeWriteMultipleCoils, slaveID, pduData, RespPDULenWriteMultipleCoils)
	if err != nil {
		return fmt.Errorf("modbus: write multiple coils failed (slave %d, address %d, quantity %d): %w", slaveID, startAddress, quantity, err) // Error already wrapped by writeModbusData
	}

	// Validate echoed data in the response PDU
	respAddress := binary.BigEndian.Uint16(respPDU[1:3])
	respQuantity := binary.BigEndian.Uint16(respPDU[3:5])

	if respAddress != startAddress {
		return fmt.Errorf("modbus: write multiple coils response start address mismatch (slave %d): expected %d, got %d", slaveID, startAddress, respAddress)
	}

	if respQuantity != quantity {
		return fmt.Errorf("modbus: write multiple coils response quantity mismatch (slave %d): expected %d, got %d", slaveID, quantity, respQuantity)
	}

	return nil
}

// WriteMultipleRegisters writes multiple registers to the Modbus device.
func (h *ModbusHandler) WriteMultipleRegisters(slaveID uint16, startAddress uint16, values []uint16) error {
	quantity := uint16(len(values))
	byteCount := quantity * 2 // Each register is 2 bytes

	// Build PDU data part (startAddress + quantity + byteCount + registerData)
	pduData := make([]byte, 5+byteCount)
	binary.BigEndian.PutUint16(pduData[0:2], startAddress)
	binary.BigEndian.PutUint16(pduData[2:4], quantity)
	pduData[4] = byte(byteCount) // Byte count field

	// Pack register values into bytes
	for i, val := range values {
		binary.BigEndian.PutUint16(pduData[5+2*i:5+2*i+2], val)
	}

	// Use generic write helper
	respPDU, err := h.writeModbusData(FuncCodeWriteMultipleRegisters, slaveID, pduData, RespPDULenWriteMultipleRegisters)
	if err != nil {
		return fmt.Errorf("modbus: write multiple registers failed (slave %d, address %d, quantity %d): %w", slaveID, startAddress, quantity, err) // Error already wrapped by writeModbusData
	}

	// Validate echoed data in the response PDU
	respAddress := binary.BigEndian.Uint16(respPDU[1:3])
	respQuantity := binary.BigEndian.Uint16(respPDU[3:5])

	if respAddress != startAddress {
		return fmt.Errorf("modbus: write multiple registers response start address mismatch (slave %d): expected %d, got %d", slaveID, startAddress, respAddress)
	}

	if respQuantity != quantity {
		return fmt.Errorf("modbus: write multiple registers response quantity mismatch (slave %d): expected %d, got %d", slaveID, quantity, respQuantity)
	}

	return nil
}

// ReadCustomData sends a request with a custom function code and returns the response PDU payload.
// Note: This method assumes the request PDU data structure starts with Address (2 bytes) and Quantity/Length (2 bytes).
// It also assumes the response PDU structure is similar to standard read operations:
// [0] Function Code (1 byte)
// [1] Byte Count (1 byte)
// [2...] Data Payload (Byte Count bytes)
// These assumptions might NOT work correctly for all custom function codes.
func (h *ModbusHandler) ReadCustomData(funcCode uint16, slaveID uint16, startAddress, quantity uint16) ([]byte, error) {
	// Build PDU data part (assuming address + quantity/length structure)
	pduData := make([]byte, 4)
	binary.BigEndian.PutUint16(pduData[0:2], startAddress)
	binary.BigEndian.PutUint16(pduData[2:4], quantity) // Using quantity for the 4th byte, may represent length

	// Build the full request PDU
	reqPDU, err := buildRequestPDU(uint8(funcCode), pduData) // Assumes buildRequestPDU exists
	if err != nil {
		if h.logger != nil {
			fmt.Fprintf(h.logger, "modbus: Error building request PDU for custom func %02X (slave %d): %v", funcCode, slaveID, err)
		}
		return nil, fmt.Errorf("modbus: failed to build request PDU for custom func %02X (slave %d): %w", funcCode, slaveID, err)
	}

	// Send request and receive response
	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return nil, fmt.Errorf("modbus: send/receive failed for custom func %02X (slave %d): %w", funcCode, slaveID, err)
	}

	// Validate response function code
	if len(respPDU) == 0 || respPDU[0] != uint8(funcCode) {
		return nil, fmt.Errorf("modbus: unexpected function code in response for custom func %02X (slave %d): got %d", funcCode, slaveID, respPDU[0])
	}

	// !!! Assume response structure includes function code + byte count + data payload !!!
	// !!! This assumption may NOT be valid for all custom function codes !!!
	if len(respPDU) < 2 {
		return nil, fmt.Errorf("modbus: invalid response length for custom func %02X (slave %d): expected at least 2 bytes, got %d. Note: Assumes standard read-like response structure", funcCode, slaveID, len(respPDU))
	}
	byteCount := int(respPDU[1])
	if len(respPDU) != 2+byteCount {
		return nil, fmt.Errorf("modbus: invalid response data length for custom func %02X (slave %d): expected %d bytes, got %d. Note: Assumes standard read-like response structure", funcCode, slaveID, byteCount, len(respPDU)-2)
	}

	// Return the raw data payload part (after func code and byte count)
	return respPDU[2:], nil
}

// WriteCustomData sends a write request with a custom function code and data.
// Note: This method assumes the request PDU data structure starts with Address (2 bytes)
// followed by the data payload. It puts the data length (uint16) before the data payload.
// It also assumes a minimal response structure, typically just the function code.
// These assumptions might NOT work correctly for all custom function codes.
func (h *ModbusHandler) WriteCustomData(funcCode uint16, slaveID uint16, startAddress uint16, data []byte) error {
	// Build PDU data part (assuming start address + length + data structure)
	pduData := make([]byte, 4+len(data)) // Address (2) + Length (2) + Data (len(data))
	binary.BigEndian.PutUint16(pduData[0:2], startAddress)
	binary.BigEndian.PutUint16(pduData[2:4], uint16(len(data))) // Assuming quantity/length goes here
	copy(pduData[4:], data)

	// Build the full request PDU
	reqPDU, err := buildRequestPDU(uint8(funcCode), pduData) // Assumes buildRequestPDU exists
	if err != nil {
		if h.logger != nil {
			fmt.Fprintf(h.logger, "modbus: Error building request PDU for custom write func %02X (slave %d): %v", funcCode, slaveID, err)
		}
		return fmt.Errorf("modbus: failed to build request PDU for custom write func %02X (slave %d): %w", funcCode, slaveID, err)
	}

	// Send request and receive response
	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return fmt.Errorf("modbus: send/receive failed for custom write func %02X (slave %d): %w", funcCode, slaveID, err)
	}

	// Validate response function code
	if len(respPDU) == 0 || respPDU[0] != uint8(funcCode) {
		return fmt.Errorf("modbus: unexpected function code in response for custom write func %02X (slave %d): got %d", funcCode, slaveID, respPDU[0])
	}

	// !!! Assume a minimal response structure, typically just the function code !!!
	// !!! Actual response structure may vary for custom functions !!!
	if len(respPDU) != 1 {
		// Log the actual response bytes for debugging custom functions
		if h.logger != nil {
			fmt.Fprintf(h.logger, "modbus: Warning: Unexpected response length for custom write func %02X (slave %d): expected 1 byte, got %d. Response: % X", funcCode, slaveID, len(respPDU), respPDU)
		}
		return fmt.Errorf("modbus: unexpected response length for custom write func %02X (slave %d): expected 1 byte, got %d", funcCode, slaveID, len(respPDU))
	}

	return nil
}

// ReadDeviceIdentity reads the device identity using Modbus function code 0x11.
// This function has a specific response structure and does not use the generic read helper.
func (h *ModbusHandler) ReadDeviceIdentity(slaveID uint16) (uint16, error) {
	// Build request PDU with function code 0x11 (data payload is often empty)
	var FuncCodeReadDeviceIdentity byte = 0x11
	reqPDU, err := buildRequestPDU(FuncCodeReadDeviceIdentity, nil) // Assumes buildRequestPDU handles nil payload
	if err != nil {
		if h.logger != nil {
			fmt.Fprintf(h.logger, "modbus: Error building request PDU for func %02X (slave %d): %v", FuncCodeReadDeviceIdentity, slaveID, err)
		}
		return 0, fmt.Errorf("modbus: failed to build request PDU for func %02X (slave %d): %w", FuncCodeReadDeviceIdentity, slaveID, err)
	}

	// Send request and receive response
	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, fmt.Errorf("modbus: send/receive failed for func %02X (slave %d): %w", FuncCodeReadDeviceIdentity, slaveID, err)
	}

	// Validate response function code
	if len(respPDU) == 0 || respPDU[0] != FuncCodeReadDeviceIdentity {
		return 0, fmt.Errorf("modbus: unexpected function code in response for func %02X (slave %d): got %d", FuncCodeReadDeviceIdentity, slaveID, respPDU[0])
	}

	// Validate minimum response length for FC 0x11
	// Response structure: FuncCode (1) + MEI Type (1) + Device ID Code (1) + more data...
	// Minimum is typically 3 bytes + subsequent data length specifier.
	// Based on the original code, it seems to expect at least 4 bytes: FuncCode (1) + Run Indicator (1) + Additional Data Length (1) + Actual Data...
	// Let's follow the original code's interpretation which seems non-standard FC 0x11, but specific to its use case.
	// A standard FC 0x11 response involves OBJs. The original code's parsing seems custom.
	// Sticking to the original parsing logic for device identity (returning the first byte of additional data).
	if len(respPDU) < 4 {
		return 0, fmt.Errorf("modbus: invalid response length for func %02X (slave %d): expected at least 4 bytes, got %d. Note: Parsing based on non-standard FC11 response", FuncCodeReadDeviceIdentity, slaveID, len(respPDU))
	}

	// Original code's parsing logic: Run indicator is first byte of 'additional data'
	// This seems to interpret respPDU[1] as the run indicator directly.
	// This is likely a simplified or non-standard implementation of FC 0x11.
	runIndicator := respPDU[1]
	// Additional data length is respPDU[2] in original code, also non-standard for FC11.
	// additionalDataLength := respPDU[2] // Not used in return

	// Return the run indicator as the device identity, as per original code's logic
	return uint16(runIndicator), nil
}

// ReadExceptionStatus reads the exception status using Modbus function code 0x07.
// This function has a specific response structure and does not use the generic read helper.
func (h *ModbusHandler) ReadExceptionStatus(slaveID uint16) (string, error) {
	// Build request PDU with function code 0x07 (data payload is nil)
	reqPDU, err := buildRequestPDU(FuncCodeReadExceptionStatus, nil) // Assumes buildRequestPDU handles nil payload
	if err != nil {
		if h.logger != nil {
			fmt.Fprintf(h.logger, "modbus: Error building request PDU for func %02X (slave %d): %v", FuncCodeReadExceptionStatus, slaveID, err)
		}
		return "", fmt.Errorf("modbus: failed to build request PDU for func %02X (slave %d): %w", FuncCodeReadExceptionStatus, slaveID, err)
	}

	// Send request and receive response
	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return "", fmt.Errorf("modbus: send/receive failed for func %02X (slave %d): %w", FuncCodeReadExceptionStatus, slaveID, err)
	}

	// Validate response function code
	if len(respPDU) == 0 || respPDU[0] != FuncCodeReadExceptionStatus {
		return "", fmt.Errorf("modbus: unexpected function code in response for func %02X (slave %d): got %d", FuncCodeReadExceptionStatus, slaveID, respPDU[0])
	}

	// Validate response length for FC 0x07
	// Standard response structure: FuncCode (1) + Status Byte (1) = 2 bytes
	if len(respPDU) != RespPDULenReadExceptionStatus {
		return "", fmt.Errorf("modbus: invalid response length for func %02X (slave %d): expected %d bytes, got %d", FuncCodeReadExceptionStatus, slaveID, RespPDULenReadExceptionStatus, len(respPDU))
	}

	// The exception status is typically a byte representing internal device status.
	// You might need to define specific meanings for different status codes based on device documentation.
	statusByte := respPDU[1]
	return fmt.Sprintf("Exception Status: 0x%02X", statusByte), nil
}

func (h *ModbusHandler) sendAndReceive(slaveID uint8, reqPDU []byte) ([]byte, error) {
	// Log the request details (optional)
	if h.logger != nil {
		// Assuming reqPDU starts with the function code after buildRequestPDU
		funcCode := uint8(0)
		if len(reqPDU) > 0 {
			funcCode = reqPDU[0]
		}
		// Log PDU data excluding the function code if it's a standard structure
		pduDataLog := reqPDU
		if len(reqPDU) > 0 {
			pduDataLog = reqPDU[1:]
		}
		if h.mode == "tcp" {
			RemoteAddr := h.tcpTransporter.conn.RemoteAddr().String()
			fmt.Fprintf(h.logger, "modbus tcp: Sending request to slave %d, func %02X, PDU data: % X, RemoteAddr: %s", slaveID, funcCode, pduDataLog, RemoteAddr)
		}
		if h.mode == "rtu" {
			fmt.Fprintf(h.logger, "modbus rtu: Sending request to slave %d, func %02X, PDU data: % X", slaveID, funcCode, pduDataLog)
		}
	}

	// Send the request PDU
	var err error
	if h.mode == "rtu" {
		err = h.rtuTransporter.Send(slaveID, reqPDU) // Assumes Transporter.Send adds SlaveID and CRC
	} else if h.mode == "tcp" {
		err = h.tcpTransporter.Send(h.transmissionID, slaveID, reqPDU) // Assumes Transporter.Send adds SlaveID and CRC
	}
	if err != nil {
		// Log and wrap the transport error
		if h.logger != nil {
			if h.mode == "tcp" {
				RemoteAddr := h.tcpTransporter.conn.RemoteAddr().String()
				fmt.Fprintf(h.logger, "modbus tcp: Error sending request to slave %d: %v, RemoteAddr: %s", slaveID, err, RemoteAddr)
			}
			if h.mode == "rtu" {
				fmt.Fprintf(h.logger, "modbus rtu: Error sending request to slave %d: %v", slaveID, err)
			}
		}
		return nil, fmt.Errorf("modbus: rtu transport send failed (slave %d): %w", slaveID, err)
	}
	// var transactionID uint16
	var respSlaveID uint8
	var respPDU []byte
	if h.mode == "rtu" {
		respSlaveID, respPDU, err = h.rtuTransporter.Receive()

	} else if h.mode == "tcp" {
		_, respSlaveID, respPDU, err = h.tcpTransporter.Receive()
	}
	if err != nil {
		// Log and wrap the transport error
		if h.logger != nil {
			if h.mode == "tcp" {
				RemoteAddr := h.tcpTransporter.conn.RemoteAddr().String()
				fmt.Fprintf(h.logger, "modbus tcp: Error receiving response from slave %d: %v, RemoteAddr: %s", slaveID, err, RemoteAddr)
			}
			if h.mode == "rtu" {
				fmt.Fprintf(h.logger, "modbus rtu: Error receiving response from slave %d: %v", slaveID, err)
			}
		}
		return nil, fmt.Errorf("modbus: rtu transport receive failed (slave %d): %w", slaveID, err)
	}
	// Log the received response details (optional)
	if h.logger != nil {
		fmt.Fprintf(h.logger, "modbus: Received response from slave %d, PDU: % X", respSlaveID, respPDU)
	}
	// Validate the received slave ID
	if respSlaveID != slaveID {
		err = fmt.Errorf("modbus: response slave ID mismatch: expected %d, got %d", slaveID, respSlaveID)
		if h.logger != nil {
			if h.mode == "tcp" {
				RemoteAddr := h.tcpTransporter.conn.RemoteAddr().String()
				fmt.Fprintf(h.logger, "modbus tcp: Error response slave ID mismatch (slave %d): %v, RemoteAddr: %s", slaveID, err, RemoteAddr)
			}
			if h.mode == "rtu" {
				fmt.Fprintf(h.logger, "modbus rtu: Error response slave ID mismatch (slave %d): %v", slaveID, err)
			}
		}
		return nil, err
	}
	if len(respPDU) > 0 && (respPDU[0]&0x80) != 0 {
		exceptionCode := uint8(0) // Default if response is too short
		if len(respPDU) > 1 {
			exceptionCode = respPDU[1] // Exception code is in the second byte
		}
		exceptionMsg := getExceptionMessage(exceptionCode) // Assumes getExceptionMessage exists
		err = fmt.Errorf("modbus: received exception response (slave %d): code 0x%02X - %s", slaveID, exceptionCode, exceptionMsg)
		if h.logger != nil {
			if h.mode == "tcp" {
				RemoteAddr := h.tcpTransporter.conn.RemoteAddr().String()
				fmt.Fprintf(h.logger, "modbus tcp: Error received exception response (slave %d): %v, RemoteAddr: %s", slaveID, err, RemoteAddr)
			}
			if h.mode == "rtu" {
				fmt.Fprintf(h.logger, "modbus rtu: Error received exception response (slave %d): %v", slaveID, err)
			}
		}
		return nil, err
	}
	return respPDU, nil
}
