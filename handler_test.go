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
	"testing"
)

func TestGetExceptionMessage(t *testing.T) {
	testCases := []struct {
		code    uint8
		message string
	}{
		{code: 0x01, message: "Illegal function"},
		{code: 0x02, message: "Illegal data address"},
		{code: 0x03, message: "Illegal data value"},
		{code: 0x04, message: "Slave device failure"},
		{code: 0x05, message: "Acknowledge"},
		{code: 0x06, message: "Slave device busy"},
		{code: 0x08, message: "Memory parity error"},
		{code: 0x0A, message: "Gateway path unavailable"},
		{code: 0x0B, message: "Gateway target device failed to respond"},
		{code: 0xFF, message: "Unknown exception code"},
	}

	for _, tc := range testCases {
		message := getExceptionMessage(tc.code)
		if message != tc.message {
			t.Errorf("GetExceptionMessage(%#02x) returned incorrect message: got %q, expected %q", tc.code, message, tc.message)
		}
	}
}
