package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strings"
	"time"
)

// TCPHandler implements the ModbusApi interface for TCP communication.
type TCPHandler struct {
	transporter *TCPTransporter
	// You might want to manage transaction IDs if needed for more complex scenarios
	transactionIDCounter uint16
	logger               io.Writer
}

// NewTCPHandler creates a new TCPHandler with the given net.Conn and timeout.
func NewTCPHandler(conn net.Conn, timeout time.Duration, logger io.Writer) *TCPHandler {
	return &TCPHandler{
		transporter:          NewTCPTransporter(conn, timeout, logger),
		transactionIDCounter: 0,
		logger:               logger,
	}
}

// incrementTransactionID increments the transaction ID counter.
func (h *TCPHandler) incrementTransactionID() uint16 {
	h.transactionIDCounter++
	return h.transactionIDCounter
}

// ReadCoils reads the specified number of coils starting from the given address.
func (h *TCPHandler) ReadCoils(slaveID uint16, startAddress, quantity uint16) ([]bool, error) {
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
func (h *TCPHandler) ReadDiscreteInputs(slaveID uint16, startAddress, quantity uint16) ([]bool, error) {
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
func (h *TCPHandler) ReadHoldingRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error) {
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
func (h *TCPHandler) ReadInputRegisters(slaveID uint16, startAddress, quantity uint16) ([]uint16, error) {
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
func (h *TCPHandler) WriteSingleCoil(slaveID uint16, address uint16, value bool) error {
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
func (h *TCPHandler) WriteSingleRegister(slaveID uint16, address uint16, value uint16) error {
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
func (h *TCPHandler) WriteMultipleCoils(slaveID uint16, startAddress uint16, values []bool) error {
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
func (h *TCPHandler) WriteMultipleRegisters(slaveID uint16, startAddress uint16, values []uint16) error {
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
func (h *TCPHandler) ReadCustomData(funcCode uint16, slaveID uint16, startAddress, quantity uint16) ([]byte, error) {
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
func (h *TCPHandler) WriteCustomData(funcCode uint16, slaveID uint16, startAddress uint16, data []byte) error {
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
func (h *TCPHandler) ReadDeviceIdentity(slaveID uint16) (string, error) {
	// Placeholder implementation. You'll need to know the specific function code and data format.
	return "", fmt.Errorf("ReadDeviceIdentity not implemented for TCP")
}

// ReadExceptionStatus reads the exception status. This is typically done with function code 0x07.
func (h *TCPHandler) ReadExceptionStatus(slaveID uint16) (string, error) {
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

// ReadUint8 reads a uint8 value.
func (h *TCPHandler) ReadUint8(slaveID uint16, address uint16, byteOrder string) (uint8, error) {
	data, err := h.ReadHoldingRegisters(slaveID, address, 1)
	if err != nil {
		return 0, err
	}
	if len(data) != 1 {
		return 0, fmt.Errorf("unexpected number of registers returned: %d, expected 1", len(data))
	}
	return uint8(data[0] & 0xFF), nil // Assuming the lower byte is the uint8
}

// ReadUint16 reads a uint16 value.
func (h *TCPHandler) ReadUint16(slaveID uint16, address uint16, byteOrder string) (uint16, error) {
	data, err := h.ReadHoldingRegisters(slaveID, address, 1)
	if err != nil {
		return 0, err
	}
	if len(data) != 1 {
		return 0, fmt.Errorf("unexpected number of registers returned: %d, expected 1", len(data))
	}
	return data[0], nil
}

// ReadUint32 reads a uint32 value.
func (h *TCPHandler) ReadUint32(slaveID uint16, address uint16, byteOrder string) (uint32, error) {
	data, err := h.ReadHoldingRegisters(slaveID, address, 2)
	if err != nil {
		return 0, err
	}
	if len(data) != 2 {
		return 0, fmt.Errorf("unexpected number of registers returned: %d, expected 2", len(data))
	}
	if strings.ToUpper(byteOrder) == "LITTLE_ENDIAN" {
		return uint32(data[0]) | uint32(data[1])<<16, nil
	}
	return uint32(data[1]) | uint32(data[0])<<16, nil
}

// ReadUint64 reads a uint64 value.
func (h *TCPHandler) ReadUint64(slaveID uint16, address uint16, byteOrder string) (uint64, error) {
	data, err := h.ReadHoldingRegisters(slaveID, address, 4)
	if err != nil {
		return 0, err
	}
	if len(data) != 4 {
		return 0, fmt.Errorf("unexpected number of registers returned: %d, expected 4", len(data))
	}
	var val uint64
	if strings.ToUpper(byteOrder) == "LITTLE_ENDIAN" {
		val |= uint64(data[0])
		val |= uint64(data[1]) << 16
		val |= uint64(data[2]) << 32
		val |= uint64(data[3]) << 48
	} else {
		val |= uint64(data[3])
		val |= uint64(data[2]) << 16
		val |= uint64(data[1]) << 32
		val |= uint64(data[0]) << 48
	}
	return val, nil
}

// ReadInt8 reads an int8 value.
func (h *TCPHandler) ReadInt8(slaveID uint16, address uint16, byteOrder string) (int8, error) {
	u, err := h.ReadUint8(slaveID, address, byteOrder)
	if err != nil {
		return 0, err
	}
	return int8(u), nil
}

// ReadInt16 reads an int16 value.
func (h *TCPHandler) ReadInt16(slaveID uint16, address uint16, byteOrder string) (int16, error) {
	u, err := h.ReadUint16(slaveID, address, byteOrder)
	if err != nil {
		return 0, err
	}
	return int16(u), nil
}

// ReadInt32 reads an int32 value.
func (h *TCPHandler) ReadInt32(slaveID uint16, address uint16, byteOrder string) (int32, error) {
	u, err := h.ReadUint32(slaveID, address, byteOrder)
	if err != nil {
		return 0, err
	}
	return int32(u), nil
}

// ReadInt64 reads an int64 value.
func (h *TCPHandler) ReadInt64(slaveID uint16, address uint16, byteOrder string) (int64, error) {
	u, err := h.ReadUint64(slaveID, address, byteOrder)
	if err != nil {
		return 0, err
	}
	return int64(u), nil
}

// ReadFloat32 reads a float32 value.
func (h *TCPHandler) ReadFloat32(slaveID uint16, address uint16, byteOrder string) (float32, error) {
	u, err := h.ReadUint32(slaveID, address, byteOrder)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(u), nil
}

// ReadFloat64 reads a float64 value.
func (h *TCPHandler) ReadFloat64(slaveID uint16, address uint16, byteOrder string) (float64, error) {
	u, err := h.ReadUint64(slaveID, address, byteOrder)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(u), nil
}

// ReadBool reads a single boolean value (from a coil).
func (h *TCPHandler) ReadBool(slaveID uint16, address uint16, byteOrder string) (bool, error) {
	coils, err := h.ReadCoils(slaveID, address, 1)
	if err != nil {
		return false, err
	}
	if len(coils) != 1 {
		return false, fmt.Errorf("unexpected number of coils returned: %d, expected 1", len(coils))
	}
	return coils[0], nil
}

// ReadBytes reads a sequence of bytes (from holding registers).
func (h *TCPHandler) ReadBytes(slaveID uint16, address uint16, length uint16, byteOrder string) ([]byte, error) {
	registerCount := (length + 1) / 2
	registers, err := h.ReadHoldingRegisters(slaveID, address, registerCount)
	if err != nil {
		return nil, err
	}
	data := make([]byte, registerCount*2)
	for i, reg := range registers {
		binary.BigEndian.PutUint16(data[i*2:i*2+2], reg)
	}
	if length%2 != 0 {
		data = data[:length]
	}
	return data, nil // Byte order is handled by individual read functions
}

// ReadBit reads a single bit from a holding register.
func (h *TCPHandler) ReadBit(slaveID uint16, address uint16, bit uint8, byteOrder string) (bool, error) {
	if bit > 15 {
		return false, fmt.Errorf("invalid bit number: %d, must be between 0 and 15", bit)
	}
	reg, err := h.ReadHoldingRegisters(slaveID, address, 1)
	if err != nil {
		return false, err
	}
	if len(reg) != 1 {
		return false, fmt.Errorf("unexpected number of registers returned: %d, expected 1", len(reg))
	}
	return (reg[0] & (1 << bit)) != 0, nil
}

// ReadBits reads multiple bits from a sequence of holding registers.
func (h *TCPHandler) ReadBits(slaveID uint16, address uint16, startBit uint8, length uint8, byteOrder string) ([]bool, error) {
	if startBit > 15 {
		return nil, fmt.Errorf("invalid start bit: %d, must be between 0 and 15", startBit)
	}
	registerCount := (int(startBit) + int(length) + 15) / 16
	registers, err := h.ReadHoldingRegisters(slaveID, address, uint16(registerCount))
	if err != nil {
		return nil, err
	}
	bits := make([]bool, length)
	bitIndex := int(startBit)
	for i := 0; i < int(length); i++ {
		regIndex := (bitIndex) / 16
		bitInReg := (bitIndex) % 16
		if regIndex < len(registers) {
			if (registers[regIndex] & (1 << bitInReg)) != 0 {
				bits[i] = true
			}
		}
		bitIndex++
	}
	return bits, nil
}

// ReadString reads a string from holding registers with the specified encoding.
func (h *TCPHandler) ReadString(slaveID uint16, address uint16, length uint16, encoding string, byteOrder string) (string, error) {
	byteCount := length
	registerCount := (byteCount + 1) / 2
	registers, err := h.ReadHoldingRegisters(slaveID, address, registerCount)
	if err != nil {
		return "", err
	}
	data := make([]byte, registerCount*2)
	for i, reg := range registers {
		binary.BigEndian.PutUint16(data[i*2:i*2+2], reg)
	}
	data = data[:byteCount]

	var str string
	switch strings.ToUpper(encoding) {
	case "ASCII":
		str = string(data)
	default:
		return "", fmt.Errorf("unsupported encoding: %s", encoding)
	}
	return str, nil
}

// sendAndReceive sends a PDU and receives the response over the TCP transporter.
func (h *TCPHandler) sendAndReceive(slaveID uint8, reqPDU []byte) ([]byte, error) {
	transactionID := h.incrementTransactionID()
	err := h.transporter.Send(transactionID, slaveID, reqPDU)
	if err != nil {
		return nil, err
	}

	respTransactionID, respSlaveID, respPDU, err := h.transporter.Receive()
	if err != nil {
		return nil, err
	}

	if respTransactionID != transactionID {
		return nil, fmt.Errorf("response transaction ID mismatch: expected %d, got %d", transactionID, respTransactionID)
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

// Close closes the underlying TCP connection.
func (h *TCPHandler) Close() error {
	return h.transporter.Close()
}
