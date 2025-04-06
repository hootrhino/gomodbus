//go:build windows && cgo
// Copyright 2014 Quoc-Viet Nguyen. All rights reserved.
// This software may be modified and distributed under the terms
// of the BSD license.  See the LICENSE file for details.
//go:build windows && cgo
// +build windows,cgo

package main

import (
	"bufio"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const port = "COM4"

func main() {
	// Convert port name to UTF-16
	portPtr, err := windows.UTF16PtrFromString(port)
	if err != nil {
		fmt.Printf("Failed to convert port to UTF-16: %v\n", err)
		return
	}

	// Open the serial port
	handle, err := windows.CreateFile(portPtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,   // No sharing
		nil, // Default security attributes
		windows.OPEN_EXISTING,
		0, // No special flags
		0)
	if err != nil {
		fmt.Printf("Failed to open port: %v\n", err)
		return
	}
	defer windows.CloseHandle(handle)
	fmt.Printf("Handle created: %v\n", handle)

	// Configure the serial port
	var dcb windows.DCB
	dcb.DCBlength = uint32(unsafe.Sizeof(dcb))
	dcb.BaudRate = 9600
	dcb.ByteSize = 8
	dcb.StopBits = windows.ONESTOPBIT
	dcb.Parity = windows.NOPARITY

	if err := windows.SetCommState(handle, &dcb); err != nil {
		fmt.Printf("Failed to set comm state: %v\n", err)
		return
	}
	fmt.Println("Comm state set successfully")

	// Set timeouts
	var timeouts windows.COMMTIMEOUTS
	timeouts.ReadIntervalTimeout = 1000
	timeouts.ReadTotalTimeoutMultiplier = 0
	timeouts.ReadTotalTimeoutConstant = 1000
	timeouts.WriteTotalTimeoutMultiplier = 0
	timeouts.WriteTotalTimeoutConstant = 1000

	if err := windows.SetCommTimeouts(handle, &timeouts); err != nil {
		fmt.Printf("Failed to set comm timeouts: %v\n", err)
		return
	}
	fmt.Println("Comm timeouts set successfully")

	// Write data to the serial port
	dataToWrite := []byte("abc")
	var bytesWritten uint32
	err = windows.WriteFile(handle, dataToWrite, &bytesWritten, nil)
	if err != nil {
		fmt.Printf("Failed to write to port: %v\n", err)
		return
	}
	fmt.Printf("Successfully wrote %d bytes\n", bytesWritten)

	// Wait for user input before reading
	fmt.Println("Press Enter when ready for reading...")
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	// Read data from the serial port
	readBuffer := make([]byte, 512)
	var bytesRead uint32
	err = windows.ReadFile(handle, readBuffer, &bytesRead, nil)
	if err != nil {
		fmt.Printf("Failed to read from port: %v\n", err)
		return
	}
	fmt.Printf("Received %d bytes: %x\n", bytesRead, readBuffer[:bytesRead])

	fmt.Println("Closing handle")
}
