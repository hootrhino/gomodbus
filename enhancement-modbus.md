# Modbus API

## 1. **ReadCoils**
- **Description**: Reads the status of multiple coils.
- **Parameters**:
  - `slaveID uint16`: The Modbus slave device ID.
  - `startAddress uint16`: The starting address for reading.
  - `quantity uint16`: The number of coils to read.
- **Return Values**:
  - `[]bool`: A boolean array representing the status of each coil.
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    coils, err := modbusClient.ReadCoils(1, 0, 8)
    if err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Coils:", coils)  // Example output: [true, false, true, ...]
    ```

---

## 2. **ReadDiscreteInputs**
- **Description**: Reads the status of multiple discrete inputs.
- **Parameters**:
  - `slaveID uint16`: The Modbus slave device ID.
  - `startAddress uint16`: The starting address for reading.
  - `quantity uint16`: The number of discrete inputs to read.
- **Return Values**:
  - `[]bool`: A boolean array representing the status of each discrete input.
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    inputs, err := modbusClient.ReadDiscreteInputs(1, 0, 8)
    if err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Discrete Inputs:", inputs)  // Example output: [true, false, true, ...]
    ```

---

## 3. **ReadHoldingRegisters**
- **Description**: Reads the values of multiple holding registers.
- **Parameters**:
  - `slaveID uint16`: The Modbus slave device ID.
  - `startAddress uint16`: The starting address for reading.
  - `quantity uint16`: The number of holding registers to read.
- **Return Values**:
  - `[]uint16`: An array of unsigned 16-bit integers representing the values of the holding registers.
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    registers, err := modbusClient.ReadHoldingRegisters(1, 0, 2)
    if err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Holding Registers:", registers)  // Example output: [1234, 5678]
    ```

---

## 4. **ReadInputRegisters**
- **Description**: Reads the values of multiple input registers.
- **Parameters**:
  - `slaveID uint16`: The Modbus slave device ID.
  - `startAddress uint16`: The starting address for reading.
  - `quantity uint16`: The number of input registers to read.
- **Return Values**:
  - `[]uint16`: An array of unsigned 16-bit integers representing the values of the input registers.
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    inputRegisters, err := modbusClient.ReadInputRegisters(1, 0, 2)
    if err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Input Registers:", inputRegisters)  // Example output: [2345, 6789]
    ```

---

## 5. **WriteSingleCoil**
- **Description**: Writes the state of a single coil.
- **Parameters**:
  - `slaveID uint16`: The Modbus slave device ID.
  - `address uint16`: The address of the coil to write.
  - `value bool`: The state to write to the coil (`true` or `false`).
- **Return Values**:
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    err := modbusClient.WriteSingleCoil(1, 0, true)
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println("Successfully wrote single coil.")
    }
    ```

---

## 6. **WriteSingleRegister**
- **Description**: Writes the value of a single register.
- **Parameters**:
  - `slaveID uint16`: The Modbus slave device ID.
  - `address uint16`: The address of the register to write.
  - `value uint16`: The value to write to the register.
- **Return Values**:
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    err := modbusClient.WriteSingleRegister(1, 0, 1234)
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println("Successfully wrote single register.")
    }
    ```

---

## 7. **ReadCustomData**
- **Description**: Reads custom data using a specified function code.
- **Parameters**:
  - `funcCode uint16`: The Modbus function code.
  - `slaveID uint16`: The Modbus slave device ID.
  - `startAddress uint16`: The starting address for reading.
  - `quantity uint16`: The length of data to read.
- **Return Values**:
  - `[]byte`: A byte array representing the custom data read.
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    data, err := modbusClient.ReadCustomData(0x03, 1, 0, 2)
    if err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Printf("Custom Data: %v\n", data)  // Example output: [1, 2]
    ```

---

## 8. **WriteCustomData**
- **Description**: Writes custom data using a specified function code.
- **Parameters**:
  - `funcCode uint16`: The Modbus function code.
  - `slaveID uint16`: The Modbus slave device ID.
  - `startAddress uint16`: The starting address for writing.
  - `data []byte`: The data to write.
- **Return Values**:
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    err := modbusClient.WriteCustomData(0x10, 1, 0, []byte{0x01, 0x02})
    if err != nil {
        fmt.Println("Error:", err)
    } else {
        fmt.Println("Successfully wrote custom data.")
    }
    ```

---

## 9. **ReadDeviceIdentity**
- **Description**: Reads the identity of the Modbus device.
- **Parameters**:
  - `slaveID uint16`: The Modbus slave device ID.
- **Return Values**:
  - `string`: The identity of the Modbus device.
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    identity, err := modbusClient.ReadDeviceIdentity(1)
    if err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Device Identity:", identity)  // Example output: "Modbus Device 123"
    ```

---

## 10. **ReadExceptionStatus**
- **Description**: Reads the exception status of the Modbus device.
- **Parameters**:
  - `slaveID uint16`: The Modbus slave device ID.
- **Return Values**:
  - `string`: The exception status.
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    status, err := modbusClient.ReadExceptionStatus(1)
    if err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Exception Status:", status)  // Example output: "No Exception"
    ```

---

## 11. **ReadUint8**
- **Description**: Reads a single 8-bit unsigned integer.
- **Parameters**:
  - `slaveID uint16`: The Modbus slave device ID.
  - `address uint16`: The address to read.
  - `byteOrder string`: The byte order, either `"BIG"` or `"LITTLE"`.
- **Return Values**:
  - `uint8`: The 8-bit unsigned integer value.
  - `error`: An error message, if an error occurs.
- **Example**:
    ```go
    value, err := modbusClient.ReadUint8(1, 0, "BIG")
    if err != nil {
        fmt.Println("Error:", err)
    }
    fmt.Println("Uint8 Value:", value)  // Example output: 255
    ```