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
	"net"
	"time"
)

// RtuOverTCPTransporter enables Modbus RTU frames to be sent over a TCP connection.
type RtuOverTCPTransporter struct {
	conn     net.Conn
	timeout  time.Duration
	packager *RTUPackager
	config   TCPTransporterConfig
}

// NewRtuOverTCPTransporter creates a new RtuOverTCPTransporter using RTUPackager.
func NewRtuOverTCPTransporter(conn net.Conn, config TCPTransporterConfig) *RtuOverTCPTransporter {
	return &RtuOverTCPTransporter{
		conn:     conn,
		config:   config,
		timeout:  config.Timeout,
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
	slaveID, pdu, err = t.packager.Unpack(raw)
	if err != nil {
		return 0, nil, err
	}
	return slaveID, pdu, nil
}

// Close closes the TCP connection.
func (t *RtuOverTCPTransporter) Close() error {
	return t.conn.Close()
}
