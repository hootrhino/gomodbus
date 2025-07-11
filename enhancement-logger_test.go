package modbus

import (
	"fmt"
	"os"
	"testing"
)

func TestLogger(t *testing.T) {

	// Example usage with os.Stdout
	loggerStdout := NewSimpleLogger(nil, LevelDebug, "TEST")
	defer loggerStdout.Close()

	loggerStdout.Write([]byte("DEBUG: This is a debug message"))
	loggerStdout.Write([]byte("INFO: This is an info message"))
	loggerStdout.Write([]byte("WARNING: This is a warning message"))
	loggerStdout.Write([]byte("ERROR: This is an error message"))
	loggerStdout.Write([]byte("This is a default info message")) // No prefix

	loggerStdout.SetLevel(LevelWarning)
	fmt.Println("\n--- After setting level to WARNING ---")
	loggerStdout.Write([]byte("DEBUG: This debug message will be filtered"))
	loggerStdout.Write([]byte("WARNING: This warning message will be shown"))
	loggerStdout.Write([]byte("ERROR: This error message will be shown"))

	// Example usage with a file output
	file, err := os.Create("app.log")
	if err != nil {
		fmt.Println("Error creating log file:", err)
		return
	}
	loggerFile := NewSimpleLogger(file, LevelInfo, "TEST")
	defer loggerFile.Close()

	loggerFile.Write([]byte("INFO: Logging to file"))
	loggerFile.Write([]byte("ERROR: An error occurred in file"))

	err = loggerFile.SetLevelFromString("debug")
	if err != nil {
		fmt.Println("Error setting level:", err)
	} else {
		fmt.Println("\n--- After setting file logger level to DEBUG via string ---")
		loggerFile.Write([]byte("DEBUG: This debug message will be logged to file"))
	}

	err = loggerFile.SetLevelFromString("INVALID")
	if err != nil {
		fmt.Println("Error setting level:", err)
	}
}
