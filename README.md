
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

## Development Guide

### **1. DeviceRegister**
The `DeviceRegister` struct is a core component of this library. It represents a Modbus register with metadata and provides methods for encoding, decoding, and interpreting register values.

#### **DeviceRegister Fields**
| Field          | Type      | Description                                                                     |
| -------------- | --------- | ------------------------------------------------------------------------------- |
| `Tag`          | `string`  | A unique identifier or label for the register.                                  |
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
This project is based on goburrow/modbus, available at [this link](https://github.com/goburrow/modbus). Thank you for your valuable contribution to the developer community.

- [Modbus Specifications](http://www.modbus.org/specs.php)
- [Modbus Protocol Description](https://www.modbustools.com/modbus.html)
```
