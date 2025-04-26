package modbus

import (
	"fmt"
	"strings"
)

// buildRequestPDU constructs a Modbus request PDU.
// It takes the function code and the data payload as input.
func buildRequestPDU(functionCode uint8, data []byte) ([]byte, error) {
	pdu := make([]byte, 1+len(data))
	pdu[0] = functionCode
	copy(pdu[1:], data)
	return pdu, nil
}

// getExceptionMessage returns a human-readable message for a Modbus exception code.
func getExceptionMessage(exceptionCode uint8) string {
	switch exceptionCode {
	case 0x01:
		return "Illegal function"
	case 0x02:
		return "Illegal data address"
	case 0x03:
		return "Illegal data value"
	case 0x04:
		return "Slave device failure"
	case 0x05:
		return "Acknowledge"
	case 0x06:
		return "Slave device busy"
	case 0x08:
		return "Memory parity error"
	case 0x0A:
		return "Gateway path unavailable"
	case 0x0B:
		return "Gateway target device failed to respond"
	default:
		return "Unknown exception code"
	}
}

// CRC16 calculates the Modbus CRC16 checksum.
func CRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if (crc & 0x0001) != 0 {
				crc >>= 1
				crc ^= 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return ((crc & 0xFF) << 8) | ((crc >> 8) & 0xFF)
}

// formatHEX formats a byte slice into a pretty-printed hex dump with aligned byte indices.
// Each line contains 8 bytes.
func formatPrintHEX(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, b := range data {
		if i > 0 {
			builder.WriteByte(' ')
		}
		fmt.Fprintf(&builder, "%02X[%02d]", b, i)
	}
	return builder.String()
}

// Helper function to convert byte order.
// convertByteOrder converts byte order based on the specified byte order string.
func convertByteOrder(data []byte, byteOrder string) []byte {
	switch byteOrder {
	case "A":
		if len(data) >= 1 {
			return data[:1]
		}
	case "AB":
		if len(data) >= 2 {
			return data[:2]
		}
	case "BA":
		if len(data) >= 2 {
			return []byte{data[1], data[0]}
		}
	case "ABCD":
		if len(data) >= 4 {
			return data[:4]
		}
	case "DCBA":
		if len(data) >= 4 {
			return []byte{data[3], data[2], data[1], data[0]}
		}
	case "BADC":
		if len(data) >= 4 {
			return []byte{data[1], data[0], data[3], data[2]}
		}
	case "CDAB":
		if len(data) >= 4 {
			return []byte{data[2], data[3], data[0], data[1]}
		}
	case "ABCDEFGH":
		if len(data) >= 8 {
			return data[:8]
		}
	case "HGFEDCBA":
		if len(data) >= 8 {
			return []byte{data[7], data[6], data[5], data[4], data[3], data[2], data[1], data[0]}
		}
	case "BADCFEHG":
		if len(data) >= 8 {
			return []byte{data[1], data[0], data[3], data[2], data[5], data[4], data[7], data[6]}
		}
	case "GHEFCDAB":
		if len(data) >= 8 {
			return []byte{data[6], data[7], data[4], data[5], data[2], data[3], data[0], data[1]}
		}
	}
	return data
}
