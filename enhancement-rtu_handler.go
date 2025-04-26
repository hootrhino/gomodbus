package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// RTUHandler implements the ModbusApi interface for RTU communication.
type RTUHandler struct {
	logger      io.Writer
	transporter *RTUTransporter
}

// NewRTUHandler creates a new RTUHandler with the given serial port and timeout.
func NewRTUHandler(port io.ReadWriteCloser, timeout time.Duration, logger io.Writer) *RTUHandler {
	return &RTUHandler{
		logger:      logger,
		transporter: NewRTUTransporter(port, timeout, logger),
	}
}

// ReadCoils reads the specified number of coils starting from the given address.
func (h *RTUHandler) ReadCoils(slaveID uint16, startAddress, quantity uint16) ([]bool, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], startAddress)
	binary.BigEndian.PutUint16(pdu[2:4], quantity)

	reqPDU, err := buildRequestPDU(FuncCodeReadCoils, pdu)
	if err != nil {
		return nil, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return nil, err
	}

	if respPDU[0] != FuncCodeReadCoils {
		return nil, fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeReadCoils)
	}

	byteCount := int(respPDU[1])
	if len(respPDU) != 2+byteCount {
		return nil, fmt.Errorf("invalid response length: expected %d bytes, got %d", 2+byteCount, len(respPDU))
	}

	coils := make([]bool, quantity)
	for i := 0; i < int(quantity); i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if byteIndex < byteCount {
			if (respPDU[2+byteIndex] & (1 << bitIndex)) != 0 {
				coils[i] = true
			}
		}
	}

	return coils, nil
}

// ReadDiscreteInputs reads the specified number of discrete inputs starting from the given address.
func (h *RTUHandler) ReadDiscreteInputs(slaveID uint16, startAddress, quantity uint16) ([]bool, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], startAddress)
	binary.BigEndian.PutUint16(pdu[2:4], quantity)

	reqPDU, err := buildRequestPDU(FuncCodeReadDiscreteInputs, pdu)
	if err != nil {
		return nil, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return nil, err
	}

	if respPDU[0] != FuncCodeReadDiscreteInputs {
		return nil, fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeReadDiscreteInputs)
	}

	byteCount := int(respPDU[1])
	if len(respPDU) != 2+byteCount {
		return nil, fmt.Errorf("invalid response length: expected %d bytes, got %d", 2+byteCount, len(respPDU))
	}

	inputs := make([]bool, quantity)
	for i := 0; i < int(quantity); i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if byteIndex < byteCount {
			if (respPDU[2+byteIndex] & (1 << bitIndex)) != 0 {
				inputs[i] = true
			}
		}
	}

	return inputs, nil
}

// ReadHoldingRegisters reads the specified number of holding registers starting from the given address.
func (h *RTUHandler) ReadHoldingRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], startAddress)
	binary.BigEndian.PutUint16(pdu[2:4], quantity)

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return nil, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return nil, err
	}

	if respPDU[0] != FuncCodeReadHoldingRegisters {
		return nil, fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeReadHoldingRegisters)
	}

	byteCount := int(respPDU[1])
	if len(respPDU) != 2+byteCount || byteCount%2 != 0 {
		return nil, fmt.Errorf("invalid response length: expected even number of bytes after count, got %d", byteCount)
	}

	registerCount := byteCount / 2
	registers := make([]uint16, registerCount)
	for i := 0; i < registerCount; i++ {
		registers[i] = binary.BigEndian.Uint16(respPDU[2+2*i : 2+2*i+2])
	}

	return registers, nil
}

// ReadInputRegisters reads the specified number of input registers starting from the given address.
func (h *RTUHandler) ReadInputRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], startAddress)
	binary.BigEndian.PutUint16(pdu[2:4], quantity)

	reqPDU, err := buildRequestPDU(FuncCodeReadInputRegisters, pdu)
	if err != nil {
		return nil, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return nil, err
	}

	if respPDU[0] != FuncCodeReadInputRegisters {
		return nil, fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeReadInputRegisters)
	}

	byteCount := int(respPDU[1])
	if len(respPDU) != 2+byteCount || byteCount%2 != 0 {
		return nil, fmt.Errorf("invalid response length: expected even number of bytes after count, got %d", byteCount)
	}

	registerCount := byteCount / 2
	registers := make([]uint16, registerCount)
	for i := 0; i < registerCount; i++ {
		registers[i] = binary.BigEndian.Uint16(respPDU[2+2*i : 2+2*i+2])
	}

	return registers, nil
}

// WriteSingleCoil writes a single coil to the Modbus device.
func (h *RTUHandler) WriteSingleCoil(slaveID uint16, address uint16, value bool) error {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	if value {
		binary.BigEndian.PutUint16(pdu[2:4], 0xFF00) // ON
	} else {
		binary.BigEndian.PutUint16(pdu[2:4], 0x0000) // OFF
	}

	reqPDU, err := buildRequestPDU(FuncCodeWriteSingleCoil, pdu)
	if err != nil {
		return err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return err
	}

	if respPDU[0] != FuncCodeWriteSingleCoil {
		return fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeWriteSingleCoil)
	}

	if len(respPDU) != 5 { // Function Code (1) + Address (2) + Value (2)
		return fmt.Errorf("invalid response length: expected 5 bytes, got %d", len(respPDU))
	}

	respAddress := binary.BigEndian.Uint16(respPDU[1:3])
	respValue := binary.BigEndian.Uint16(respPDU[3:5])

	if respAddress != address {
		return fmt.Errorf("response address mismatch: expected %d, got %d", address, respAddress)
	}

	var expectedValue uint16
	if value {
		expectedValue = 0xFF00
	}

	if respValue != expectedValue {
		return fmt.Errorf("response value mismatch: expected 0x%04X, got 0x%04X", expectedValue, respValue)
	}

	return nil
}

// WriteSingleRegister writes a single register to the Modbus device.
func (h *RTUHandler) WriteSingleRegister(slaveID uint16, address uint16, value uint16) error {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], value)

	reqPDU, err := buildRequestPDU(FuncCodeWriteSingleRegister, pdu)
	if err != nil {
		return err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return err
	}

	if respPDU[0] != FuncCodeWriteSingleRegister {
		return fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeWriteSingleRegister)
	}

	if len(respPDU) != 5 { // Function Code (1) + Address (2) + Value (2)
		return fmt.Errorf("invalid response length: expected 5 bytes, got %d", len(respPDU))
	}

	respAddress := binary.BigEndian.Uint16(respPDU[1:3])
	respValue := binary.BigEndian.Uint16(respPDU[3:5])

	if respAddress != address {
		return fmt.Errorf("response address mismatch: expected %d, got %d", address, respAddress)
	}

	if respValue != value {
		return fmt.Errorf("response value mismatch: expected %d, got %d", value, respValue)
	}

	return nil
}

// WriteMultipleCoils writes multiple coils to the Modbus device.
func (h *RTUHandler) WriteMultipleCoils(slaveID uint16, startAddress uint16, values []bool) error {
	quantity := uint16(len(values))
	byteCount := (quantity + 7) / 8
	pdu := make([]byte, 5+byteCount)
	binary.BigEndian.PutUint16(pdu[0:2], startAddress)
	binary.BigEndian.PutUint16(pdu[2:4], quantity)
	pdu[4] = byte(byteCount)

	for i := 0; i < int(quantity); i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		if values[i] {
			pdu[5+byteIndex] |= (1 << bitIndex)
		}
	}

	reqPDU, err := buildRequestPDU(FuncCodeWriteMultipleCoils, pdu)
	if err != nil {
		return err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return err
	}

	if respPDU[0] != FuncCodeWriteMultipleCoils {
		return fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeWriteMultipleCoils)
	}

	if len(respPDU) != 5 { // Function Code (1) + Address (2) + Quantity (2)
		return fmt.Errorf("invalid response length: expected 5 bytes, got %d", len(respPDU))
	}

	respAddress := binary.BigEndian.Uint16(respPDU[1:3])
	respQuantity := binary.BigEndian.Uint16(respPDU[3:5])

	if respAddress != startAddress {
		return fmt.Errorf("response start address mismatch: expected %d, got %d", startAddress, respAddress)
	}

	if respQuantity != quantity {
		return fmt.Errorf("response quantity mismatch: expected %d, got %d", quantity, respQuantity)
	}

	return nil
}

// WriteMultipleRegisters writes multiple registers to the Modbus device.
func (h *RTUHandler) WriteMultipleRegisters(slaveID uint16, startAddress uint16, values []uint16) error {
	quantity := uint16(len(values))
	byteCount := quantity * 2
	pdu := make([]byte, 5+byteCount)
	binary.BigEndian.PutUint16(pdu[0:2], startAddress)
	binary.BigEndian.PutUint16(pdu[2:4], quantity)
	pdu[4] = byte(byteCount)

	for i, val := range values {
		binary.BigEndian.PutUint16(pdu[5+2*i:5+2*i+2], val)
	}

	reqPDU, err := buildRequestPDU(FuncCodeWriteMultipleRegisters, pdu)
	if err != nil {
		return err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return err
	}

	if respPDU[0] != FuncCodeWriteMultipleRegisters {
		return fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeWriteMultipleRegisters)
	}

	if len(respPDU) != 5 { // Function Code (1) + Address (2) + Quantity (2)
		return fmt.Errorf("invalid response length: expected 5 bytes, got %d", len(respPDU))
	}

	respAddress := binary.BigEndian.Uint16(respPDU[1:3])
	respQuantity := binary.BigEndian.Uint16(respPDU[3:5])

	if respAddress != startAddress {
		return fmt.Errorf("response start address mismatch: expected %d, got %d", startAddress, respAddress)
	}

	if respQuantity != quantity {
		return fmt.Errorf("response quantity mismatch: expected %d, got %d", quantity, respQuantity)
	}

	return nil
}

// ReadCustomData reads custom data from the Modbus device.
func (h *RTUHandler) ReadCustomData(funcCode uint16, slaveID uint16, startAddress, quantity uint16) ([]byte, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], startAddress)
	binary.BigEndian.PutUint16(pdu[2:4], quantity)

	reqPDU, err := buildRequestPDU(uint8(funcCode), pdu)
	if err != nil {
		return nil, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return nil, err
	}

	if respPDU[0] != uint8(funcCode) {
		return nil, fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], funcCode)
	}

	if len(respPDU) < 1 {
		return nil, fmt.Errorf("invalid response length")
	}

	// The structure of custom data response depends on the function code.
	// We return the raw data part of the PDU (excluding the function code).
	return respPDU[1:], nil
}

// WriteCustomData writes custom data to the Modbus device.
func (h *RTUHandler) WriteCustomData(funcCode uint16, slaveID uint16, startAddress uint16, data []byte) error {
	pdu := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(pdu[0:2], startAddress)
	binary.BigEndian.PutUint16(pdu[2:4], uint16(len(data))) // Assuming quantity/length is needed
	copy(pdu[4:], data)

	reqPDU, err := buildRequestPDU(uint8(funcCode), pdu)
	if err != nil {
		return err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return err
	}

	if respPDU[0] != uint8(funcCode) {
		return fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], funcCode)
	}

	// Assuming a minimal response for write operations (just the function code).
	// The actual response structure might vary based on the custom function.
	if len(respPDU) != 1 {
		// Consider logging the actual response for debugging custom functions.
		return fmt.Errorf("unexpected response length for custom write function: %d", len(respPDU))
	}

	return nil
}

// ReadDeviceIdentity reads the device identity. This is typically done with a specific function code (e.g., 0x2B).
// The implementation here is a placeholder and would require the specific details of the function code and response format.
func (h *RTUHandler) ReadDeviceIdentity(slaveID uint16) (string, error) {
	// Placeholder implementation. You'll need to know the specific function code and data format.
	return "", fmt.Errorf("ReadDeviceIdentity not implemented for RTU")
}

// ReadExceptionStatus reads the exception status. This is typically done with function code 0x07.
func (h *RTUHandler) ReadExceptionStatus(slaveID uint16) (string, error) {
	reqPDU, err := buildRequestPDU(FuncCodeReadExceptionStatus, nil)
	if err != nil {
		return "", err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return "", err
	}

	if respPDU[0] != FuncCodeReadExceptionStatus {
		return "", fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeReadExceptionStatus)
	}

	if len(respPDU) != 2 {
		return "", fmt.Errorf("invalid response length for Read Exception Status: expected 2 bytes, got %d", len(respPDU))
	}

	// The exception status is typically a byte representing internal device status.
	// You might need to define specific meanings for different status codes.
	return fmt.Sprintf("Exception Status: 0x%02X", respPDU[1]), nil
}

// sendAndReceive sends a PDU and receives the response over the RTU transporter.
func (h *RTUHandler) sendAndReceive(slaveID uint8, reqPDU []byte) ([]byte, error) {
	err := h.transporter.Send(slaveID, reqPDU)
	if err != nil {
		return nil, err
	}
	respSlaveID, respPDU, err := h.transporter.Receive()
	if err != nil {
		return nil, err
	}
	if respSlaveID != slaveID {
		return nil, fmt.Errorf("response slave ID mismatch: expected %d, got %d", slaveID, respSlaveID)
	}

	// Check for Modbus exception
	if len(respPDU) > 0 && (respPDU[0]&0x80) != 0 {
		exceptionCode := respPDU[1]
		return nil, fmt.Errorf("modbus exception: code 0x%02X - %s", exceptionCode, getExceptionMessage(exceptionCode))
	}

	return respPDU, nil
}

// ReadUint8 reads a single unsigned 8-bit value from the Modbus device.
func (h *RTUHandler) ReadUint8(slaveID uint16, address uint16, byteOrder string) (uint8, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 1) // Read 1 byte

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 3 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the byte in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return data[0], nil
}

// ReadUint16 reads a single unsigned 16-bit value from the Modbus device.
func (h *RTUHandler) ReadUint16(slaveID uint16, address uint16, byteOrder string) (uint16, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 1) // Read 1 register (2 bytes)

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 3 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the value in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return binary.BigEndian.Uint16(data), nil
}

// ReadUint32 reads a single unsigned 32-bit value from the Modbus device.
func (h *RTUHandler) ReadUint32(slaveID uint16, address uint16, byteOrder string) (uint32, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 2) // Read 2 registers (4 bytes)

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 5 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the value in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return binary.BigEndian.Uint32(data), nil
}

// ReadUint64 reads a single unsigned 64-bit value from the Modbus device.
func (h *RTUHandler) ReadUint64(slaveID uint16, address uint16, byteOrder string) (uint64, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 4) // Read 4 registers (8 bytes)

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 7 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the value in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return binary.BigEndian.Uint64(data), nil
}

// ReadInt8 reads a single signed 8-bit value from the Modbus device.
func (h *RTUHandler) ReadInt8(slaveID uint16, address uint16, byteOrder string) (int8, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 1) // Read 1 byte

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 3 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the byte in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return int8(data[0]), nil
}

// ReadInt16 reads a single signed 16-bit value from the Modbus device.
func (h *RTUHandler) ReadInt16(slaveID uint16, address uint16, byteOrder string) (int16, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 1) // Read 1 register (2 bytes)

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 3 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the value in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return int16(binary.BigEndian.Uint16(data)), nil
}

// ReadInt32 reads a single signed 32-bit value from the Modbus device.
func (h *RTUHandler) ReadInt32(slaveID uint16, address uint16, byteOrder string) (int32, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 2) // Read 2 registers (4 bytes)

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 5 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the value in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return int32(binary.BigEndian.Uint32(data)), nil
}

// ReadInt64 reads a single signed 64-bit value from the Modbus device.
func (h *RTUHandler) ReadInt64(slaveID uint16, address uint16, byteOrder string) (int64, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 4) // Read 4 registers (8 bytes)

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 7 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the value in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return int64(binary.BigEndian.Uint64(data)), nil
}

// ReadFloat32 reads a single 32-bit floating point value from the Modbus device.
func (h *RTUHandler) ReadFloat32(slaveID uint16, address uint16, byteOrder string) (float32, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 2) // Read 2 registers (4 bytes)

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 5 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the value in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return float32(binary.BigEndian.Uint32(data)), nil
}

// ReadFloat64 reads a single 64-bit floating point value from the Modbus device.
func (h *RTUHandler) ReadFloat64(slaveID uint16, address uint16, byteOrder string) (float64, error) {
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)
	binary.BigEndian.PutUint16(pdu[2:4], 4) // Read 4 registers (8 bytes)

	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return 0, err
	}

	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return 0, err
	}

	// Validate response length
	if len(respPDU) < 7 {
		return 0, fmt.Errorf("invalid response length")
	}

	// Convert and return the value in correct byte order
	data := convertByteOrder(respPDU[3:], byteOrder)
	return float64(binary.BigEndian.Uint64(data)), nil
}

// ReadBytes reads the specified number of bytes from the Modbus device starting from the given address,
// and returns the bytes in the specified byte order (BIG or LITTLE).
func (h *RTUHandler) ReadBytes(slaveID uint16, address uint16, quantity uint16, endian string) ([]byte, error) {
	// Create the PDU (Protocol Data Unit) for the request
	pdu := make([]byte, 4)
	binary.BigEndian.PutUint16(pdu[0:2], address)  // Starting address
	binary.BigEndian.PutUint16(pdu[2:4], quantity) // Number of bytes to read

	// Build the full request PDU using the function code for reading holding registers
	reqPDU, err := buildRequestPDU(FuncCodeReadHoldingRegisters, pdu)
	if err != nil {
		return nil, err
	}

	// Send the request and receive the response PDU
	respPDU, err := h.sendAndReceive(uint8(slaveID), reqPDU)
	if err != nil {
		return nil, err
	}

	// Ensure the response function code is as expected
	if respPDU[0] != FuncCodeReadHoldingRegisters {
		return nil, fmt.Errorf("unexpected function code in response: %d, expected %d", respPDU[0], FuncCodeReadHoldingRegisters)
	}

	// Extract byte count and check the validity of the response length
	byteCount := int(respPDU[1])
	if len(respPDU) != 2+byteCount {
		return nil, fmt.Errorf("invalid response length: expected %d bytes, got %d", 2+byteCount, len(respPDU))
	}

	// Extract the data bytes from the response
	data := respPDU[2:]

	// Handle byte order (endian) adjustments
	switch endian {
	case "BIG":
		// No changes needed for BIG Endian, as it's already in that format
	case "LITTLE":
		// Reverse the byte order to LITTLE Endian
		for i := 0; i < len(data)/2; i++ {
			data[i], data[len(data)-1-i] = data[len(data)-1-i], data[i]
		}
	default:
		return nil, fmt.Errorf("unsupported endian type: %s", endian)
	}

	// Return the processed byte array
	return data, nil
}
