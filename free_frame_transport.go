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
	"io"
	"net"
	"sync"
	"time"
)

// FreeFrameTransport supports sending and receiving arbitrary binary frames over various transports (serial, TCP, UDP).
type FreeFrameTransport struct {
	conn         io.ReadWriteCloser // Can be a serial port, TCP, or UDP connection
	readTimeout  time.Duration
	writeTimeout time.Duration
	mu           sync.RWMutex
}

// NewFreeFrameTransport creates a new FreeFrameTransport with the given connection and timeouts.
func NewFreeFrameTransport(conn io.ReadWriteCloser, readTimeout, writeTimeout time.Duration) *FreeFrameTransport {
	return &FreeFrameTransport{
		conn:         conn,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

// WriteRaw writes raw bytes to the underlying connection.
func (t *FreeFrameTransport) WriteRaw(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(data) == 0 {
		return fmt.Errorf("cannot write empty data")
	}
	if c, ok := t.conn.(net.Conn); ok && t.writeTimeout > 0 {
		_ = c.SetWriteDeadline(time.Now().Add(t.writeTimeout))
		defer c.SetWriteDeadline(time.Time{})
	}
	n, err := t.conn.Write(data)
	if err != nil {
		return fmt.Errorf("write failed after %d bytes: %v", n, err)
	}
	if n != len(data) {
		return fmt.Errorf("partial write: expected %d bytes, wrote %d", len(data), n)
	}
	return nil
}

// ReadRaw reads up to maxLen bytes from the underlying connection.
func (t *FreeFrameTransport) ReadRaw(maxLen int) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if maxLen <= 0 {
		maxLen = 1024
	}
	buf := make([]byte, maxLen)
	if c, ok := t.conn.(net.Conn); ok && t.readTimeout > 0 {
		_ = c.SetReadDeadline(time.Now().Add(t.readTimeout))
		defer c.SetReadDeadline(time.Time{})
	}
	n, err := t.conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read failed: %v", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("no data read")
	}
	return buf[:n], nil
}

// Send sends a free frame (arbitrary binary data).
func (t *FreeFrameTransport) Send(frame []byte) error {
	return t.WriteRaw(frame)
}

// Receive receives a free frame (arbitrary binary data).
func (t *FreeFrameTransport) Receive(maxLen int) ([]byte, error) {
	return t.ReadRaw(maxLen)
}

// Close closes the underlying connection.
func (t *FreeFrameTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn == nil {
		return nil
	}
	err := t.conn.Close()
	t.conn = nil
	return err
}

// IsConnected returns true if the connection is still open.
func (t *FreeFrameTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.conn != nil
}

// SetReadTimeout sets the read timeout for the transport.
func (t *FreeFrameTransport) SetReadTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.readTimeout = timeout
}

// SetWriteTimeout sets the write timeout for the transport.
func (t *FreeFrameTransport) SetWriteTimeout(timeout time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.writeTimeout = timeout
}
