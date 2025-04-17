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

## New Features

### **1. DeviceRegister**
The `DeviceRegister` struct represents a Modbus register with metadata.

#### **Example**
```go
register := DeviceRegister{
	Tag:         "Temperature",
	Alias:       "Temp Sensor",
	Function:    3,
	SlaverId:    1,
	ReadAddress: 100,
	ReadQuantity: 1,
	DataType:    "float32",
	DataOrder:   "ABCD",
	Weight:      1.0,
}
fmt.Println(register.String())
```

---

### **2. Load Registers from CSV**
You can load Modbus registers from a CSV file using the `LoadRegisterFromCSV` function.

#### **CSV Example**
```csv
Tag,Alias,Function,SlaveId,Address,Frequency,Quantity,DataType,BitMask,DataOrder,Weight
"Temperature","Temp Sensor",3,1,100,1000,1,"float32",0,"ABCD",1.0
"Pressure","Pressure Sensor",3,1,101,1000,1,"uint16",0,"AB",1.0
```

#### **Code Example**
```go
registers, err := LoadRegisterFromCSV("registers.csv")
if err != nil {
	log.Fatalf("Failed to load registers: %v", err)
}
for _, reg := range registers {
	fmt.Println(reg)
}
```

---

### **3. RegisterManager**
The `RegisterManager` handles grouped registers, callbacks, and data processing.

#### **Example**
```go
manager := NewRegisterManager(client, "TCP", 100)

// Set callbacks
manager.SetOnReadCallback(func(registers []DeviceRegister) {
	fmt.Println("Read registers:", registers)
})
manager.SetOnErrorCallback(func(err error) {
	fmt.Println("Error:", err)
})

// Load registers
registers := []DeviceRegister{
	{Tag: "Temp", Alias: "Temperature", Function: 3, SlaverId: 1, ReadAddress: 100, ReadQuantity: 1, DataType: "float32"},
	{Tag: "Pressure", Alias: "Pressure Sensor", Function: 3, SlaverId: 1, ReadAddress: 101, ReadQuantity: 1, DataType: "uint16"},
}
manager.LoadRegisters(registers)

// Start processing
manager.Start()

// Stop the manager when done
manager.Stop()
```

---

### **4. Grouping Registers**
Registers can be grouped based on address continuity using `GroupDeviceRegister`.

#### **Example**
```go
registers := []DeviceRegister{
	{ReadAddress: 100},
	{ReadAddress: 101},
	{ReadAddress: 200},
	{ReadAddress: 201},
}
groups := GroupDeviceRegister(registers)
for i, group := range groups {
	fmt.Printf("Group %d: %v\n", i+1, group)
}
```

---

### **5. Decode Register Values**
The `DecodeValue` method decodes raw Modbus register values into various data types.

#### **Example**
```go
register := DeviceRegister{
	DataType: "float32",
	Value:    [8]byte{0x42, 0x48, 0x00, 0x00}, // 50.0 in IEEE 754
}
decoded, err := register.DecodeValue()
if err != nil {
	fmt.Println("Error decoding value:", err)
} else {
	fmt.Printf("Decoded value: %v (float64: %f)\n", decoded.AsType, decoded.Float64)
}
```

---

### **6. Testing**
The library includes comprehensive test cases. Run the tests using:
```bash
go test -v ./...
```

#### **Example Test**
```go
func Test_float32FromBits(t *testing.T) {
	tests := []struct {
		name   string
		bits   uint32
		expect float32
	}{
		{name: "positive_float", bits: 0x42480000, expect: 50.0},
		{name: "negative_float", bits: 0xC2480000, expect: -50.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := float32FromBits(tt.bits)
			if math.Abs(float64(result-tt.expect)) > 0.0001 {
				t.Errorf("float32FromBits(%#x) = %f, want %f", tt.bits, result, tt.expect)
			}
		})
	}
}
```

---

## **Advanced Features**

### **1. Concurrent Reading**
The `ReadGroupedData` method supports concurrent reading of grouped registers.

#### **Example**
```go
manager.ReadGroupedData()
```

---

### **2. Byte Reordering**
The `reorderBytes` function reorders bytes based on the `DataOrder` field.

#### **Example**
```go
data := [8]byte{0x12, 0x34, 0x56, 0x78}
reordered := reorderBytes(data, "DCBA")
fmt.Printf("Reordered bytes: %v\n", reordered)
```

---

## **Error Handling**
- Use `SetOnErrorCallback` to handle errors during data processing.
- Ensure proper validation of CSV files before loading.

---

## References
This project is based on goburrow/modbus, available at [this link](https://github.com/goburrow/modbus). Thank you for your valuable contribution to the developer community.

-   [Modbus Specifications and Implementation Guides](http://www.modbus.org/specs.php)
