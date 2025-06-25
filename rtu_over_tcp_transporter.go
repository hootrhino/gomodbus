package modbus

import (
	"net"
	"time"
)

// RtuOverTCPTransporter enables Modbus RTU frames to be sent over a TCP connection.
type RtuOverTCPTransporter struct {
	conn     net.Conn
	timeout  time.Duration
	packager *RTUPackager
}

// NewRtuOverTCPTransporter creates a new RtuOverTCPTransporter using RTUPackager.
func NewRtuOverTCPTransporter(conn net.Conn, timeout time.Duration) *RtuOverTCPTransporter {
	return &RtuOverTCPTransporter{
		conn:     conn,
		timeout:  timeout,
		packager: NewRTUPackager(),
	}
}

// WriteRaw writes a raw RTU frame to the TCP connection.
func (t *RtuOverTCPTransporter) WriteRaw(frame []byte) error {
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return err
	}
	defer t.conn.SetDeadline(time.Time{})
	_, err := t.conn.Write(frame)
	return err
}

// ReadRaw reads a raw RTU frame from the TCP connection.
func (t *RtuOverTCPTransporter) ReadRaw() ([]byte, error) {
	buffer := make([]byte, 256)
	if err := t.conn.SetDeadline(time.Now().Add(t.timeout)); err != nil {
		return nil, err
	}
	defer t.conn.SetDeadline(time.Time{})
	n, err := t.conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

// Send packs and sends a Modbus RTU frame (SlaveID + PDU + CRC) over TCP.
func (t *RtuOverTCPTransporter) Send(slaveID uint8, pdu []byte) error {
	frame, err := t.packager.Pack(slaveID, pdu)
	if err != nil {
		return err
	}
	return t.WriteRaw(frame)
}

// Receive reads a Modbus RTU response frame from TCP and unpacks it using RTUPackager.
func (t *RtuOverTCPTransporter) Receive() (slaveID uint8, pdu []byte, err error) {
	raw, err := t.ReadRaw()
	if err != nil {
		return 0, nil, err
	}
	return t.packager.Unpack(raw)
}

// Close closes the TCP connection.
func (t *RtuOverTCPTransporter) Close() error {
	return t.conn.Close()
}
