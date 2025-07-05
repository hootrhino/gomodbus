package modbus

import (
	"bytes"
	"testing"
)

func TestRTUPackager_PackUnpack(t *testing.T) {
	p := NewRTUPackager()
	slaveID := uint8(1)
	pdu := []byte{0x03, 0x00, 0x00, 0x00, 0x01} // Function 3, address 0x0000, quantity 1

	frame, err := p.Pack(slaveID, pdu)
	if err != nil {
		t.Fatalf("Pack failed: %v", err)
	}

	// CRC 校验
	if !p.VerifyCRC(frame) {
		t.Fatalf("VerifyCRC failed on packed frame")
	}

	// Unpack
	gotSlave, gotPDU, err := p.Unpack(frame)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}
	if gotSlave != slaveID {
		t.Errorf("Unpack slaveID mismatch: got %d, want %d", gotSlave, slaveID)
	}
	if !bytes.Equal(gotPDU, pdu) {
		t.Errorf("Unpack PDU mismatch: got %v, want %v", gotPDU, pdu)
	}
}

func TestRTUPackager_VerifyCRC_Invalid(t *testing.T) {
	p := NewRTUPackager()
	frame := []byte{0x01, 0x03, 0x02, 0x12, 0x34, 0x00, 0x00} // 错误CRC
	if p.VerifyCRC(frame) {
		t.Error("VerifyCRC should fail for invalid CRC")
	}
}

func TestRTUPackager_RepairFrame(t *testing.T) {
	p := NewRTUPackager()
	frame := []byte{0x01, 0x03, 0x02, 0x12, 0x34, 0x00, 0x00} // 错误CRC
	repaired, err := p.RepairFrame(frame)
	if err != nil {
		t.Fatalf("RepairFrame failed: %v", err)
	}
	if !p.VerifyCRC(repaired) {
		t.Error("VerifyCRC failed after repair")
	}
}

func TestRTUPackager_CompareCRCMethods(t *testing.T) {
	p := NewRTUPackager()
	data := []byte{0x01, 0x03, 0x02, 0x12, 0x34}
	tableCRC, directCRC, equal := p.CompareCRCMethods(data)
	if !equal {
		t.Errorf("CRC methods mismatch: table=%04X, direct=%04X", tableCRC, directCRC)
	}
}

func TestRTUPackager_Pack_Invalid(t *testing.T) {
	p := NewRTUPackager()
	_, err := p.Pack(0, []byte{0x03, 0x00}) // slaveID 0非法
	if err == nil {
		t.Error("Pack should fail for invalid slaveID")
	}
	_, err = p.Pack(1, []byte{}) // 空PDU
	if err == nil {
		t.Error("Pack should fail for empty PDU")
	}
	_, err = p.Pack(1, make([]byte, 254)) // 超长PDU
	if err == nil {
		t.Error("Pack should fail for too long PDU")
	}
}

func TestRTUPackager_Unpack_ShortFrame(t *testing.T) {
	p := NewRTUPackager()
	_, _, err := p.Unpack([]byte{0x01, 0x03, 0x00}) // 少于4字节
	if err == nil {
		t.Error("Unpack should fail for short frame")
	}
}
