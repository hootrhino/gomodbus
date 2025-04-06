// Copyright 2014 Quoc-Viet Nguyen. All rights reserved.
// This software may be modified and distributed under the terms
// of the BSD license. See the LICENSE file for details.

// Test serial communication in Win32
// Compile with: gcc commw32.c -o commw32.exe

#include <stdlib.h>
#include <stdio.h>
#include <windows.h>

static const wchar_t *port = L"COM4";

// Function to print the last error message
static void printLastError()
{
	wchar_t lpBuffer[256] = L"?";
	FormatMessage(
		FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
		NULL,
		GetLastError(),
		MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT),
		lpBuffer,
		sizeof(lpBuffer) / sizeof(wchar_t) - 1,
		NULL);
	wprintf(L"Error: %ls\n", lpBuffer);
}

// Function to configure the serial port
static int configureSerialPort(HANDLE handle)
{
	DCB dcb = {0};
	dcb.DCBlength = sizeof(dcb);
	dcb.BaudRate = CBR_9600;
	dcb.ByteSize = 8;
	dcb.StopBits = ONESTOPBIT;
	dcb.Parity = NOPARITY;
	dcb.fTXContinueOnXoff = 1; // No software handshaking
	dcb.fOutX = 0;
	dcb.fInX = 0;
	dcb.fBinary = 1;	   // Binary mode
	dcb.fAbortOnError = 0; // No blocking on errors

	if (!SetCommState(handle, &dcb))
	{
		printf("Failed to set comm state.\n");
		printLastError();
		return 0;
	}
	printf("Comm state configured successfully.\n");
	return 1;
}

// Function to configure timeouts for the serial port
static int configureTimeouts(HANDLE handle)
{
	COMMTIMEOUTS timeouts = {0};
	timeouts.ReadIntervalTimeout = 1000; // Timeout between characters (ms)
	timeouts.ReadTotalTimeoutMultiplier = 0;
	timeouts.ReadTotalTimeoutConstant = 1000;
	timeouts.WriteTotalTimeoutMultiplier = 0;
	timeouts.WriteTotalTimeoutConstant = 1000;

	if (!SetCommTimeouts(handle, &timeouts))
	{
		printf("Failed to set comm timeouts.\n");
		printLastError();
		return 0;
	}
	printf("Comm timeouts configured successfully.\n");
	return 1;
}

int main()
{
	HANDLE handle;
	DWORD bytesTransferred = 0;
	char readBuffer[512];
	int i;

	// Open the serial port
	handle = CreateFile(
		port,
		GENERIC_READ | GENERIC_WRITE,
		0,			   // No sharing
		NULL,		   // Default security attributes
		OPEN_EXISTING, // Open existing port
		0,			   // No special flags
		NULL		   // No template file
	);

	if (handle == INVALID_HANDLE_VALUE)
	{
		printf("Failed to open port: %s\n", port);
		printLastError();
		return 1;
	}
	printf("Serial port %s opened successfully.\n", port);

	// Configure the serial port
	if (!configureSerialPort(handle))
	{
		CloseHandle(handle);
		return 1;
	}

	// Configure timeouts
	if (!configureTimeouts(handle))
	{
		CloseHandle(handle);
		return 1;
	}

	// Write data to the serial port
	const char *dataToWrite = "abc";
	if (!WriteFile(handle, dataToWrite, 3, &bytesTransferred, NULL))
	{
		printf("Failed to write to port.\n");
		printLastError();
		CloseHandle(handle);
		return 1;
	}
	printf("Successfully wrote %lu bytes to the port.\n", bytesTransferred);

	// Wait for user input before reading
	printf("Press Enter when ready to read data...");
	getchar();

	// Read data from the serial port
	if (!ReadFile(handle, readBuffer, sizeof(readBuffer), &bytesTransferred, NULL))
	{
		printf("Failed to read from port.\n");
		printLastError();
		CloseHandle(handle);
		return 1;
	}
	printf("Successfully read %lu bytes from the port.\n", bytesTransferred);

	// Print received data in hexadecimal format
	printf("Received data:\n");
	for (i = 0; i < (int)bytesTransferred; ++i)
	{
		printf("%02X ", (unsigned char)readBuffer[i]);
	}
	printf("\n");

	// Close the serial port
	CloseHandle(handle);
	printf("Serial port closed.\n");

	return 0;
}
