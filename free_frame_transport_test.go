package modbus

import (
	"bytes"
	"io"
	"testing"
	"time"
)

// mockConn is a simple in-memory ReadWriteCloser for testing.
type mockConn struct {
	io.Reader
	io.Writer
	closed bool
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func TestFreeFrameTransport_WriteReadRaw(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := &mockConn{Reader: buf, Writer: buf}
	transport := NewFreeFrameTransport(conn, 0, 0)

	data := []byte{0x01, 0x02, 0x03}
	err := transport.WriteRaw(data)
	if err != nil {
		t.Fatalf("WriteRaw failed: %v", err)
	}

	// Reset buffer for reading test
	readBuf := bytes.NewBuffer(data)
	conn.Reader = readBuf

	out, err := transport.ReadRaw(3)
	if err != nil {
		t.Fatalf("ReadRaw failed: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("ReadRaw returned %v, want %v", out, data)
	}
}

func TestFreeFrameTransport_SendReceive(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := &mockConn{Reader: buf, Writer: buf}
	transport := NewFreeFrameTransport(conn, 0, 0)

	frame := []byte{0xAA, 0xBB, 0xCC}
	if err := transport.Send(frame); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Prepare buffer for Receive
	readBuf := bytes.NewBuffer(frame)
	conn.Reader = readBuf

	out, err := transport.Receive(3)
	if err != nil {
		t.Fatalf("Receive failed: %v", err)
	}
	if !bytes.Equal(out, frame) {
		t.Errorf("Receive returned %v, want %v", out, frame)
	}
}

func TestFreeFrameTransport_Close_IsConnected(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := &mockConn{Reader: buf, Writer: buf}
	transport := NewFreeFrameTransport(conn, 0, 0)

	if !transport.IsConnected() {
		t.Error("IsConnected should be true after creation")
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if transport.IsConnected() {
		t.Error("IsConnected should be false after Close")
	}
}

func TestFreeFrameTransport_TimeoutSetters(t *testing.T) {
	buf := &bytes.Buffer{}
	conn := &mockConn{Reader: buf, Writer: buf}
	transport := NewFreeFrameTransport(conn, 0, 0)

	transport.SetReadTimeout(2 * time.Second)
	transport.SetWriteTimeout(3 * time.Second)
	if transport.readTimeout != 2*time.Second {
		t.Errorf("SetReadTimeout failed, got %v", transport.readTimeout)
	}
	if transport.writeTimeout != 3*time.Second {
		t.Errorf("SetWriteTimeout failed, got %v", transport.writeTimeout)
	}
}
