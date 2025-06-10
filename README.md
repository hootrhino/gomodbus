
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

---

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

---

## ModbusDevicePoller

`ModbusDevicePoller` is a high-level polling scheduler for gomodbus. It periodically reads register data from multiple Modbus devices in batches and dispatches the results asynchronously via callback functions.

### Features

- Manage multiple Modbus devices (TCP/RTU) simultaneously
- Automatically groups registers with continuous addresses for efficient batch reading
- Configurable polling interval
- Asynchronous data and error callbacks for easy integration
- Supports both concurrent and sequential reading

### Example Usage

```go
handler := NewModbusRTUHandler(port, 1*time.Second)
mgr := NewModbusRegisterManager(handler, 10)
mgr.LoadRegisters([]DeviceRegister{
    {Tag: "reg1", SlaverId: 1, ReadAddress: 0, ReadQuantity: 5, Function: 3},
})
mgr.SetOnData(func(data []DeviceRegister) {
    // Handle received data
})
mgr.SetOnError(func(err error) {
    // Handle error
})

poller := NewModbusDevicePoller(100 * time.Millisecond)
poller.AddManager(mgr)
poller.Start()
defer poller.Stop()
```

### Main API

- `NewModbusDevicePoller(interval time.Duration)` - Create a new poller with the specified interval
- `AddManager(mgr *ModbusRegisterManager)` - Add a register manager to the poller
- `Start()` - Start polling
- `Stop()` - Stop polling

### Test Reference

See `enhancement-poller_test.go` for integration tests, for example:

```go
func TestModbusDevicePollerWithRTU(t *testing.T) {
    // ...setup handler, manager, poller, callbacks, start and assert...
}
```

### Use Cases

- Industrial automation data acquisition
- Periodic monitoring of multiple devices
- Efficient batch reading of Modbus device data

---
## Development Guide

### **1. DeviceRegister**
The `DeviceRegister` struct is a core component of this library. It represents a Modbus register with metadata and provides methods for encoding, decoding, and interpreting register values.

#### **DeviceRegister Fields**
| Field          | Type      | Description                                                                     |
| -------------- | --------- | ------------------------------------------------------------------------------- |
| `Tag`          | `string`  | A unique identifier or label for the register.                                  |
| Type           | `string`  | Type of the register (e.g., `Metric`,  `Static`).                               |
| `Alias`        | `string`  | A human-readable name or alias for the register.                                |
| `SlaverId`     | `uint8`   | ID of the Modbus slave device.                                                  |
| `Function`     | `uint8`   | Modbus function code (e.g., 3 for Read Holding Registers).                      |
| `ReadAddress`  | `uint16`  | Address of the register in the Modbus device.                                   |
| `ReadQuantity` | `uint16`  | Number of registers to read/write.                                              |
| `DataType`     | `string`  | Data type of the register value (e.g., `uint16`, `int32`, `float32`, `string`). |
| `DataOrder`    | `string`  | Byte order for multi-byte values (e.g., `ABCD`, `DCBA`).                        |
| `BitPosition`  | `uint16`  | Bit position for bit-level operations (e.g., 0, 1, 2).                          |
| `BitMask`      | `uint16`  | Bitmask for bit-level operations (e.g., `0x01`, `0x02`).                        |
| `Weight`       | `float64` | Scaling factor for the register value.                                          |
| `Frequency`    | `uint64`  | Polling frequency in milliseconds.                                              |
| `Value`        | `[]byte`  | Raw value of the register as a byte array (variable length).                    |
| `Status`       | `string`  | Status of the register (e.g., `"OK"`, `"Error"`).                               |

---

### **2. DecodeValue**
The `DecodeValue` method converts the raw bytes in the `Value` field into a typed value based on the `DataType` and `DataOrder`.

#### **Supported Data Types**
- `bitfield`
- `bool`
- `byte`
- `uint8`, `int8`
- `uint16`, `int16`
- `uint32`, `int32`
- `float32`, `float64`
- `string`

#### **Example**
```go
register := DeviceRegister{
	DataType:    "float32",
	DataOrder:   "ABCD",
	Value:       []byte{0x42, 0x48, 0x00, 0x00}, // 50.0 in IEEE 754
	Weight:      1.0,
}
decoded, err := register.DecodeValue()
if err != nil {
	fmt.Println("Error decoding value:", err)
} else {
	fmt.Printf("Decoded value: %v (float64: %f)\n", decoded.AsType, decoded.Float64)
}
```

#### **Output**
```plaintext
Decoded value: 50 (float64: 50.000000)
```

---

### **3. Byte Reordering**
The `reorderBytes` function reorders the bytes in the `Value` field based on the `DataOrder` field.

#### **Supported Byte Orders**
| Order      | Description                         |
| ---------- | ----------------------------------- |
| `A`        | Single byte.                        |
| `AB`       | Two bytes in big-endian order.      |
| `BA`       | Two bytes in little-endian order.   |
| `ABCD`     | Four bytes in big-endian order.     |
| `DCBA`     | Four bytes in little-endian order.  |
| `BADC`     | Four bytes in word-swapped order.   |
| `ABCDEFGH` | Eight bytes in big-endian order.    |
| `HGFEDCBA` | Eight bytes in little-endian order. |

#### **Example**
```go
data := []byte{0x12, 0x34, 0x56, 0x78}
reordered := reorderBytes(data, "DCBA")
fmt.Printf("Reordered bytes: %v\n", reordered)
```

#### **Output**
```plaintext
Reordered bytes: [120 86 52 18]
```

---

### **4. Load Registers from CSV**
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

### **5. Testing**
The library includes comprehensive test cases for `DeviceRegister` and related functions. Run the tests using:
```bash
go test -v ./...
```

#### **Example Test**
```go
func Test_DeviceRegister_DecodeValue(t *testing.T) {
	register := DeviceRegister{
		DataType: "uint16",
		Value:    []byte{0x12, 0x34},
		DataOrder: "AB",
	}
	decoded, err := register.DecodeValue()
	if err != nil {
		t.Fatalf("Error decoding value: %v", err)
	}
	if decoded.Float64 != 4660 {
		t.Errorf("Expected 4660, got %f", decoded.Float64)
	}
}
```

---

### **6. Advanced Features**
#### **Concurrent Reading**
The `ReadGroupedData` method supports concurrent reading of grouped registers.

#### **Example**
```go
manager.ReadGroupedData()
```

#### **Error Handling**
- Use `SetOnErrorCallback` to handle errors during data processing.
- Ensure proper validation of CSV files before loading.

---

## References

- [Modbus Specifications](http://www.modbus.org/specs.php)
- [Modbus Protocol Description](https://www.modbustools.com/modbus.html)
```
