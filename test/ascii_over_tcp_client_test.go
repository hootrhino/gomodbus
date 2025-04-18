// Copyright 2018 xft. All rights reserved.
// This software may be modified and distributed under the terms
// of the BSD license.  See the LICENSE file for details.

package test

import (
	modbus "github.com/hootrhino/gomodbus"
	"os"
	"testing"
	"time"
)

const (
	asciiOverTCPDevice = "localhost:5020"
)

func TestASCIIOverTCPClient(t *testing.T) {
	// Diagslave does not support broadcast id.
	handler := modbus.NewASCIIOverTCPClientHandler(asciiOverTCPDevice)
	handler.SlaveId = 17
	ClientTestAll(t, modbus.NewClient(handler))
}

func TestASCIIOverTCPClientAdvancedUsage(t *testing.T) {
	handler := modbus.NewASCIIOverTCPClientHandler(asciiOverTCPDevice)
	handler.Timeout = 5 * time.Second
	handler.SlaveId = 1
	handler.Logger = modbus.NewSimpleLogger(os.Stdout, modbus.LevelDebug)
	handler.Connect()
	defer handler.Close()

	client := modbus.NewClient(handler)
	results, err := client.ReadDiscreteInputs(15, 2)
	if err != nil || results == nil {
		t.Fatal(err, results)
	}
	results, err = client.WriteMultipleRegisters(1, 2, []byte{0, 3, 0, 4})
	if err != nil || results == nil {
		t.Fatal(err, results)
	}
	results, err = client.WriteMultipleCoils(5, 10, []byte{4, 3})
	if err != nil || results == nil {
		t.Fatal(err, results)
	}
}
