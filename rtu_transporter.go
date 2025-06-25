package modbus

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// RTUTransporter handles Modbus RTU communication over a serial port.
type RTUTransporter struct {
	timeout  time.Duration
	packager *RTUPackager
	port     io.ReadWriteCloser
}

// NewRTUTransporter creates a new RTUTransporter with the given serial port and timeout.
func NewRTUTransporter(port io.ReadWriteCloser, timeout time.Duration) *RTUTransporter {
	return &RTUTransporter{
		port:     port,
		timeout:  timeout,
		packager: NewRTUPackager(),
	}
}
func (t *RTUTransporter) WriteRaw(pdu []byte) error {
	// Set timeout for write operation
	_, err := t.port.Write(pdu)
	return err
}

// ReadRaw reads a raw bytes serial port.
func (t *RTUTransporter) ReadRaw() ([]byte, error) {
	buffer := make([]byte, 512) // Adjust size as needed
	n, err := t.port.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

// Send sends a Modbus RTU PDU over the serial port.
func (t *RTUTransporter) Send(slaveID uint8, pdu []byte) error {
	frame, err := t.packager.Pack(slaveID, pdu)
	if err != nil {
		return err
	}
	_, err = t.port.Write(frame)
	return err
}

func (t *RTUTransporter) Receive() (uint8, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(t.port, header); err != nil {
		return 0, nil, fmt.Errorf("failed to read header: %v", err)
	}
	slaveID := header[0]
	functionCode := header[1]

	var frame []byte
	frame = append(frame, header...)

	var payload []byte

	switch functionCode {
	case FuncCodeReadCoils, FuncCodeReadDiscreteInputs, FuncCodeReadHoldingRegisters, FuncCodeReadInputRegisters:
		countByte := make([]byte, 1)
		if _, err := io.ReadFull(t.port, countByte); err != nil {
			return 0, nil, fmt.Errorf("failed to read byte count: %v", err)
		}
		frame = append(frame, countByte...)

		expectedDataLength := int(countByte[0])
		payload = make([]byte, expectedDataLength)
		if _, err := io.ReadFull(t.port, payload); err != nil {
			return 0, nil, fmt.Errorf("failed to read payload: %v", err)
		}
		frame = append(frame, payload...)

	case FuncCodeWriteSingleCoil, FuncCodeWriteSingleRegister, FuncCodeWriteMultipleCoils, FuncCodeWriteMultipleRegisters:
		payload = make([]byte, 4)
		if _, err := io.ReadFull(t.port, payload); err != nil {
			return 0, nil, fmt.Errorf("failed to read payload: %v", err)
		}
		frame = append(frame, payload...)

	case FuncCodeReadExceptionStatus:
		payload = make([]byte, 1)
		if _, err := io.ReadFull(t.port, payload); err != nil {
			return 0, nil, fmt.Errorf("failed to read payload: %v", err)
		}
		frame = append(frame, payload...)

	default:
		return 0, nil, fmt.Errorf("unsupported function code: %v", functionCode)
	}

	crcBytes := make([]byte, 2)
	if _, err := io.ReadFull(t.port, crcBytes); err != nil {
		return 0, nil, fmt.Errorf("failed to read CRC: %v", err)
	}
	receivedCRC := binary.BigEndian.Uint16(crcBytes)
	calculatedCRC := CRC16(frame)

	if receivedCRC != calculatedCRC {
		return 0, nil, fmt.Errorf("CRC mismatch: received %#04x, calculated %#04x, frame: % X", receivedCRC, calculatedCRC, frame)
	}

	pdu := frame[1:]
	return slaveID, pdu, nil
}

// Close closes the underlying serial port.
func (t *RTUTransporter) Close() error {
	return t.port.Close()
}
