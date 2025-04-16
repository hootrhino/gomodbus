<div style="text-align: center;">
  <h1>go-modbus</h1>
  <img src="./readme/logo.png" alt="logo" width="90px">
  <h2>Fault-tolerant, fail-fast implementation of Modbus protocol in Go.</h2>
</div>


## Supported functions
### Bit access:
*   Read Discrete Inputs
*   Read Coils
*   Write Single Coil
*   Write Multiple Coils

### 16-bit access:
*   Read Input Registers
*   Read Holding Registers
*   Write Single Register
*   Write Multiple Registers
*   Read/Write Multiple Registers
*   Mask Write Register
*   Read FIFO Queue

## Supported formats
*   TCP
*   Serial (RTU, ASCII)

## Usage
### Basic usage:
```go
// Modbus TCP
client := modbus.TCPClient("localhost:502")
// Read input register 9
results, err := client.ReadInputRegisters(8, 1)

// Modbus RTU/ASCII
// Default configuration is 19200, 8, 1, even
client = modbus.RTUClient("/dev/ttyS0")
results, err = client.ReadCoils(2, 1)
```

### Advanced usage:
Example1:
```go
// Modbus TCP
handler := modbus.NewTCPClientHandler("localhost:502")
handler.Timeout = 10 * time.Second
handler.SlaveId = 0xFF
handler.Logger = log.New(os.Stdout, "test: ", log.LstdFlags)
// Connect manually so that multiple requests are handled in one connection session
err := handler.Connect()
defer handler.Close()

client := modbus.NewClient(handler)
results, err := client.ReadDiscreteInputs(15, 2)
results, err = client.WriteMultipleRegisters(1, 2, []byte{0, 3, 0, 4})
results, err = client.WriteMultipleCoils(5, 10, []byte{4, 3})
```

Example2:

```go
// Modbus RTU/ASCII
handler := modbus.NewRTUClientHandler("/dev/ttyUSB0")
handler.BaudRate = 115200
handler.DataBits = 8
handler.Parity = "N"
handler.StopBits = 1
handler.SlaveId = 1
handler.Timeout = 5 * time.Second

err := handler.Connect()
defer handler.Close()

client := modbus.NewClient(handler)
results, err := client.ReadDiscreteInputs(15, 2)
```

### Read By Group
```go
func Test_GroupDeviceRegister(t *testing.T) {
	input := []DeviceRegister{
		{Tag: "F", Alias: "A6", SlaverId: 1, Function: 3, Address: 1, Quantity: 1},
		{Tag: "A", Alias: "A1", SlaverId: 1, Function: 3, Address: 2, Quantity: 1},
		{Tag: "B", Alias: "A2", SlaverId: 1, Function: 3, Address: 4, Quantity: 1},
		{Tag: "C", Alias: "A3", SlaverId: 1, Function: 3, Address: 5, Quantity: 1},
		{Tag: "D", Alias: "A4", SlaverId: 1, Function: 3, Address: 8, Quantity: 1},
		{Tag: "E", Alias: "A5", SlaverId: 1, Function: 3, Address: 9, Quantity: 1},
		{Tag: "G", Alias: "A7", SlaverId: 1, Function: 3, Address: 10, Quantity: 1},
	}

	{
		grouped := GroupDeviceRegister(input)
		jsonData, err := json.MarshalIndent(grouped, "", "  ")
		if err != nil {
			t.Fatalf("error marshalling result: %v", err)
		}
		t.Logf("Grouped: %s", string(jsonData))
	}
	handler := NewRTUClientHandler("COM3")
	handler.BaudRate = 9600
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.SlaveId = 1
	handler.Logger = NewSimpleLogger(os.Stdout, LevelDebug)

	err := handler.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer handler.Close()
	client := NewClient(handler)
	defer client.GetTransporter().Close()
	result := client.ReadGroupedRegisterValue(input)
	for i, group := range result {
		for j, reg := range group {
			t.Logf("======= group->%v  reg=%v  Address= %v  Tag= %v", i, j, reg.Address, reg.Tag)
		}
	}
}

```

## References

-   [Modbus Specifications and Implementation Guides](http://www.modbus.org/specs.php)
