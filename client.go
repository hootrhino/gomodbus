// Copyright 2014 Quoc-Viet Nguyen. All rights reserved.
// This software may be modified and distributed under the terms
// of the BSD license. See the LICENSE file for details.

package modbus

import (
	"encoding/binary"
	"fmt"
)

// ClientHandler is the interface that groups the Packager and Transporter methods.
type ClientHandler interface {
	Packager
	Transporter
	Type() string
	SetSlaverId(slaveId byte)
}

type client struct {
	packager    Packager
	transporter Transporter
	handler     ClientHandler
	clientType  string
}

// NewClient creates a new modbus client with given backend handler.
func NewClient(handler ClientHandler) Client {
	return &client{
		packager:    handler,
		transporter: handler,
		handler:     handler,
		clientType:  handler.Type(),
	}
}

func (mb *client) SetSlaveId(slaveId byte) {
	mb.handler.SetSlaverId(slaveId)
}
func (mb *client) GetHandlerType() string {
	return mb.handler.Type()
}

func (mb *client) Close() error {
	if mb.transporter != nil {
		return mb.transporter.Close()
	}
	return nil
}

// NewClient2 creates a new modbus client with given backend packager and transporter.
func NewClientWithTransporter(packager Packager, transporter Transporter) Client {
	return &client{packager: packager, transporter: transporter}
}

// GroupReadData reads batches of grouped registers using the provided Modbus client
func (mb *client) ReadGroupedRegisterValue(registers []DeviceRegister) [][]DeviceRegister {
	return mb.groupReadData(GroupDeviceRegisterWithUniqueAddress(registers))
}

// ReadData reads batches of grouped registers using the provided Modbus client
func (mb *client) groupReadData(registers [][]DeviceRegister) [][]DeviceRegister {
	if mb.clientType == "TCP" {
		return ReadGroupedDataConcurrently(mb, registers)
	}
	return ReadGroupedDataSequential(mb, registers)
}
func (mb *client) Type() string {
	return mb.clientType
}

// get Transporter
func (mb *client) GetTransporter() Transporter {
	return mb.transporter
}

// Request:
//
//	Function code         : 1 byte (0x01)
//	Starting address      : 2 bytes
//	Quantity of coils     : 2 bytes
//
// Response:
//
//	Function code         : 1 byte (0x01)
//	Byte count            : 1 byte
//	Coil status           : N* bytes (=N or N+1)
func (mb *client) ReadCoils(address, quantity uint16) (results []byte, err error) {
	if quantity < 1 || quantity > 2000 {
		err = fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v',", quantity, 1, 2000)
		return
	}
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeReadCoils,
		Data:         dataBlock(address, quantity),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	count := int(response.Data[0])
	length := len(response.Data) - 1
	if count != length {
		err = fmt.Errorf("modbus: response data size '%v' does not match count '%v'", length, count)
		return
	}
	results = response.Data[1:]
	return
}

// Request:
//
//	Function code         : 1 byte (0x02)
//	Starting address      : 2 bytes
//	Quantity of inputs    : 2 bytes
//
// Response:
//
//	Function code         : 1 byte (0x02)
//	Byte count            : 1 byte
//	Input status          : N* bytes (=N or N+1)
func (mb *client) ReadDiscreteInputs(address, quantity uint16) (results []byte, err error) {
	if quantity < 1 || quantity > 2000 {
		err = fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v',", quantity, 1, 2000)
		return
	}
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeReadDiscreteInputs,
		Data:         dataBlock(address, quantity),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	count := int(response.Data[0])
	length := len(response.Data) - 1
	if count != length {
		err = fmt.Errorf("modbus: response data size '%v' does not match count '%v'", length, count)
		return
	}
	results = response.Data[1:]
	return
}

// Request:
//
//	Function code         : 1 byte (0x03)
//	Starting address      : 2 bytes
//	Quantity of registers : 2 bytes
//
// Response:
//
//	Function code         : 1 byte (0x03)
//	Byte count            : 1 byte
//	Register value        : Nx2 bytes
func (mb *client) ReadHoldingRegisters(address, quantity uint16) (results []byte, err error) {
	if quantity < 1 || quantity > 125 {
		err = fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v',", quantity, 1, 125)
		return
	}
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeReadHoldingRegisters,
		Data:         dataBlock(address, quantity),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	count := int(response.Data[0])
	length := len(response.Data) - 1
	if count != length {
		err = fmt.Errorf("modbus: response data size '%v' does not match count '%v'", length, count)
		return
	}
	results = response.Data[1:]
	return
}

// Request:
//
//	Function code         : 1 byte (0x04)
//	Starting address      : 2 bytes
//	Quantity of registers : 2 bytes
//
// Response:
//
//	Function code         : 1 byte (0x04)
//	Byte count            : 1 byte
//	Input registers       : N bytes
func (mb *client) ReadInputRegisters(address, quantity uint16) (results []byte, err error) {
	if quantity < 1 || quantity > 125 {
		err = fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v',", quantity, 1, 125)
		return
	}
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeReadInputRegisters,
		Data:         dataBlock(address, quantity),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	count := int(response.Data[0])
	length := len(response.Data) - 1
	if count != length {
		err = fmt.Errorf("modbus: response data size '%v' does not match count '%v'", length, count)
		return
	}
	results = response.Data[1:]
	return
}

// Request:
//
//	Function code         : 1 byte (0x05)
//	Output address        : 2 bytes
//	Output value          : 2 bytes
//
// Response:
//
//	Function code         : 1 byte (0x05)
//	Output address        : 2 bytes
//	Output value          : 2 bytes
func (mb *client) WriteSingleCoil(address, value uint16) (results []byte, err error) {
	// The requested ON/OFF state can only be 0xFF00 and 0x0000
	if value != 0xFF00 && value != 0x0000 {
		err = fmt.Errorf("modbus: state '%v' must be either 0xFF00 (ON) or 0x0000 (OFF)", value)
		return
	}
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeWriteSingleCoil,
		Data:         dataBlock(address, value),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	// Fixed response length
	if len(response.Data) != 4 {
		err = fmt.Errorf("modbus: response data size '%v' does not match expected '%v'", len(response.Data), 4)
		return
	}
	respValue := binary.BigEndian.Uint16(response.Data)
	if address != respValue {
		err = fmt.Errorf("modbus: response address '%v' does not match request '%v'", respValue, address)
		return
	}
	results = response.Data[2:]
	respValue = binary.BigEndian.Uint16(results)
	if value != respValue {
		err = fmt.Errorf("modbus: response value '%v' does not match request '%v'", respValue, value)
		return
	}
	return
}

// Request:
//
//	Function code         : 1 byte (0x06)
//	Register address      : 2 bytes
//	Register value        : 2 bytes
//
// Response:
//
//	Function code         : 1 byte (0x06)
//	Register address      : 2 bytes
//	Register value        : 2 bytes
func (mb *client) WriteSingleRegister(address, value uint16) (results []byte, err error) {
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeWriteSingleRegister,
		Data:         dataBlock(address, value),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	// Fixed response length
	if len(response.Data) != 4 {
		err = fmt.Errorf("modbus: response data size '%v' does not match expected '%v'", len(response.Data), 4)
		return
	}
	respValue := binary.BigEndian.Uint16(response.Data)
	if address != respValue {
		err = fmt.Errorf("modbus: response address '%v' does not match request '%v'", respValue, address)
		return
	}
	results = response.Data[2:]
	respValue = binary.BigEndian.Uint16(results)
	if value != respValue {
		err = fmt.Errorf("modbus: response value '%v' does not match request '%v'", respValue, value)
		return
	}
	return
}

// Request:
//
//	Function code         : 1 byte (0x0F)
//	Starting address      : 2 bytes
//	Quantity of outputs   : 2 bytes
//	Byte count            : 1 byte
//	Outputs value         : N* bytes
//
// Response:
//
//	Function code         : 1 byte (0x0F)
//	Starting address      : 2 bytes
//	Quantity of outputs   : 2 bytes
func (mb *client) WriteMultipleCoils(address, quantity uint16, value []byte) (results []byte, err error) {
	if quantity < 1 || quantity > 1968 {
		err = fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v',", quantity, 1, 1968)
		return
	}
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeWriteMultipleCoils,
		Data:         dataBlockSuffix(value, address, quantity),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	// Fixed response length
	if len(response.Data) != 4 {
		err = fmt.Errorf("modbus: response data size '%v' does not match expected '%v'", len(response.Data), 4)
		return
	}
	respValue := binary.BigEndian.Uint16(response.Data)
	if address != respValue {
		err = fmt.Errorf("modbus: response address '%v' does not match request '%v'", respValue, address)
		return
	}
	results = response.Data[2:]
	respValue = binary.BigEndian.Uint16(results)
	if quantity != respValue {
		err = fmt.Errorf("modbus: response quantity '%v' does not match request '%v'", respValue, quantity)
		return
	}
	return
}

// Request:
//
//	Function code         : 1 byte (0x10)
//	Starting address      : 2 bytes
//	Quantity of outputs   : 2 bytes
//	Byte count            : 1 byte
//	Registers value       : N* bytes
//
// Response:
//
//	Function code         : 1 byte (0x10)
//	Starting address      : 2 bytes
//	Quantity of registers : 2 bytes
func (mb *client) WriteMultipleRegisters(address, quantity uint16, value []byte) (results []byte, err error) {
	if quantity < 1 || quantity > 123 {
		err = fmt.Errorf("modbus: quantity '%v' must be between '%v' and '%v',", quantity, 1, 123)
		return
	}
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeWriteMultipleRegisters,
		Data:         dataBlockSuffix(value, address, quantity),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	// Fixed response length
	if len(response.Data) != 4 {
		err = fmt.Errorf("modbus: response data size '%v' does not match expected '%v'", len(response.Data), 4)
		return
	}
	respValue := binary.BigEndian.Uint16(response.Data)
	if address != respValue {
		err = fmt.Errorf("modbus: response address '%v' does not match request '%v'", respValue, address)
		return
	}
	results = response.Data[2:]
	respValue = binary.BigEndian.Uint16(results)
	if quantity != respValue {
		err = fmt.Errorf("modbus: response quantity '%v' does not match request '%v'", respValue, quantity)
		return
	}
	return
}

// Request:
//
//	Function code         : 1 byte (0x16)
//	Reference address     : 2 bytes
//	AND-mask              : 2 bytes
//	OR-mask               : 2 bytes
//
// Response:
//
//	Function code         : 1 byte (0x16)
//	Reference address     : 2 bytes
//	AND-mask              : 2 bytes
//	OR-mask               : 2 bytes
func (mb *client) MaskWriteRegister(address, andMask, orMask uint16) (results []byte, err error) {
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeMaskWriteRegister,
		Data:         dataBlock(address, andMask, orMask),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	// Fixed response length
	if len(response.Data) != 6 {
		err = fmt.Errorf("modbus: response data size '%v' does not match expected '%v'", len(response.Data), 6)
		return
	}
	respValue := binary.BigEndian.Uint16(response.Data)
	if address != respValue {
		err = fmt.Errorf("modbus: response address '%v' does not match request '%v'", respValue, address)
		return
	}
	respValue = binary.BigEndian.Uint16(response.Data[2:])
	if andMask != respValue {
		err = fmt.Errorf("modbus: response AND-mask '%v' does not match request '%v'", respValue, andMask)
		return
	}
	respValue = binary.BigEndian.Uint16(response.Data[4:])
	if orMask != respValue {
		err = fmt.Errorf("modbus: response OR-mask '%v' does not match request '%v'", respValue, orMask)
		return
	}
	results = response.Data[2:]
	return
}

// Request:
//
//	Function code         : 1 byte (0x17)
//	Read starting address : 2 bytes
//	Quantity to read      : 2 bytes
//	Write starting address: 2 bytes
//	Quantity to write     : 2 bytes
//	Write byte count      : 1 byte
//	Write registers value : N* bytes
//
// Response:
//
//	Function code         : 1 byte (0x17)
//	Byte count            : 1 byte
//	Read registers value  : Nx2 bytes
func (mb *client) ReadWriteMultipleRegisters(readAddress, readQuantity, writeAddress, writeQuantity uint16, value []byte) (results []byte, err error) {
	if readQuantity < 1 || readQuantity > 125 {
		err = fmt.Errorf("modbus: quantity to read '%v' must be between '%v' and '%v',", readQuantity, 1, 125)
		return
	}
	if writeQuantity < 1 || writeQuantity > 121 {
		err = fmt.Errorf("modbus: quantity to write '%v' must be between '%v' and '%v',", writeQuantity, 1, 121)
		return
	}
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeReadWriteMultipleRegisters,
		Data:         dataBlockSuffix(value, readAddress, readQuantity, writeAddress, writeQuantity),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	count := int(response.Data[0])
	if count != (len(response.Data) - 1) {
		err = fmt.Errorf("modbus: response data size '%v' does not match count '%v'", len(response.Data)-1, count)
		return
	}
	results = response.Data[1:]
	return
}

// Request:
//
//	Function code         : 1 byte (0x18)
//	FIFO pointer address  : 2 bytes
//
// Response:
//
//	Function code         : 1 byte (0x18)
//	Byte count            : 2 bytes
//	FIFO count            : 2 bytes
//	FIFO count            : 2 bytes (<=31)
//	FIFO value register   : Nx2 bytes
func (mb *client) ReadFIFOQueue(address uint16) (results []byte, err error) {
	request := ProtocolDataUnit{
		FunctionCode: FuncCodeReadFIFOQueue,
		Data:         dataBlock(address),
	}
	response, err := mb.send(&request)
	if err != nil {
		return
	}
	if len(response.Data) < 4 {
		err = fmt.Errorf("modbus: response data size '%v' is less than expected '%v'", len(response.Data), 4)
		return
	}
	count := int(binary.BigEndian.Uint16(response.Data))
	if count != (len(response.Data) - 1) {
		err = fmt.Errorf("modbus: response data size '%v' does not match count '%v'", len(response.Data)-1, count)
		return
	}
	count = int(binary.BigEndian.Uint16(response.Data[2:]))
	if count > 31 {
		err = fmt.Errorf("modbus: fifo count '%v' is greater than expected '%v'", count, 31)
		return
	}
	results = response.Data[4:]
	return
}

// Helpers

// send sends request and checks possible exception in the response.
func (mb *client) send(request *ProtocolDataUnit) (response *ProtocolDataUnit, err error) {
	aduRequest, err := mb.packager.Encode(request)
	if err != nil {
		return
	}
	aduResponse, err := mb.transporter.Send(aduRequest)
	if err != nil {
		return
	}
	if err = mb.packager.Verify(aduRequest, aduResponse); err != nil {
		return
	}
	response, err = mb.packager.Decode(aduResponse)
	if err != nil {
		return
	}
	// Check correct function code returned (exception)
	if response.FunctionCode != request.FunctionCode {
		err = responseError(response)
		return
	}
	if len(response.Data) == 0 {
		// Empty response
		err = fmt.Errorf("modbus: response data is empty")
		return
	}
	return
}

// dataBlock creates a sequence of uint16 data.
func dataBlock(value ...uint16) []byte {
	data := make([]byte, 2*len(value))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	return data
}

// dataBlockSuffix creates a sequence of uint16 data and append the suffix plus its length.
func dataBlockSuffix(suffix []byte, value ...uint16) []byte {
	length := 2 * len(value)
	data := make([]byte, length+1+len(suffix))
	for i, v := range value {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	data[length] = uint8(len(suffix))
	copy(data[length+1:], suffix)
	return data
}

func responseError(response *ProtocolDataUnit) error {
	mbError := &ModbusError{FunctionCode: response.FunctionCode}
	if len(response.Data) > 0 {
		mbError.ExceptionCode = response.Data[0]
	}
	return mbError
}

// Request:
//
//	Function code         : 1 byte (0x2B)
//	MEI type  			  : 1 byte (0x0E)
//	Read Device ID Code	  : 1 byte
//	Object ID			  : 1 byte
//
// Response:
//
//	 Function code         : 1 byte (0x2B)
//	 MEI type  			  : 1 byte (0x0E)
//	 Read Device ID Code	  : 1 byte
//	 Conformity level	  : 1 byte
//	 More follows		  : 1 byte
//	 Next object ID		  : 1 byte
//	 Number of objects	  : 1 byte
//	 List of objects		  : <Number of objects>
//	  Object ID			  : 1
//	  Object length		  : 1
//		 Object value		  : <Object length> bytes
func (mb *client) ReadDeviceIdentification(firstExtendedID byte) (results map[byte]string, err error) {
	readDevIDCode := byte(0x01)
	objectID := byte(0x00)
	conformityLevel := byte(0x00)

	objects := make(map[byte]string)
	results = make(map[byte]string)

	// Getting basic objects (mandatory)
	for {
		conformityLevel, objectID, objects, err =
			mb.sendReadDeviceIdentification(readDevIDCode, objectID)
		if err != nil {
			return results, err
		}

		for k, v := range objects {
			results[k] = v
		}

		if len(results) >= 3 {
			break
		} else {
			if objectID == 0x00 {
				err := fmt.Errorf("modbus: mandatory device identification objects are not available")
				return results, err
			}
		}
	}

	// Getting regular and extended objects, if supported and requested by the user
	for {
		if (readDevIDCode == 0x01) && (conformityLevel&0x02) >= 0x02 {
			readDevIDCode = 0x02
			objectID = 0x00
		} else if (readDevIDCode == 0x02) && (conformityLevel&0x03) == 0x03 && firstExtendedID >= 0x80 {
			readDevIDCode = 0x03
			objectID = firstExtendedID
		} else {
			break
		}

		for {
			_, objectID, objects, err =
				mb.sendReadDeviceIdentification(readDevIDCode, objectID)
			if err != nil {
				return results, err
			}

			for k, v := range objects {
				results[k] = v
			}

			if objectID == 0x00 {
				break
			}
		}
	}

	return
}

// sendReadDeviceIdentification sends a FC43/14 request and returns the reponse after some basic checks
func (mb *client) sendReadDeviceIdentification(readDeviceIDCode byte, objectID byte) (
	conformityLevel byte, nextObjID byte, objects map[byte]string, err error) {

	objects = make(map[byte]string)

	reqData := make([]byte, 3)
	reqData[0] = MEITypeReadDeviceIdentification
	reqData[1] = readDeviceIDCode
	reqData[2] = objectID

	request := ProtocolDataUnit{
		FunctionCode: FuncCodeMEI,
		Data:         reqData,
	}

	response, err := mb.send(&request)
	if err != nil {
		return
	}

	conformityLevel = response.Data[2]
	if !((conformityLevel >= 0x01 && conformityLevel <= 0x03) ||
		(conformityLevel >= 0x81 && conformityLevel <= 0x83)) {
		err = fmt.Errorf("modbus: response conformitiy level '%v' is not valid", conformityLevel)
		return
	}

	moreFollows := response.Data[3]
	if moreFollows == 0xFF {
		nextObjID = response.Data[4]
	}

	count := response.Data[5]
	index := 6
	for i := byte(0); i < count; i++ {
		id := response.Data[index]
		length := int(response.Data[index+1])
		value := response.Data[index+2 : index+2+length]

		objects[id] = string(value)
		index += 2 + length
	}

	return
}
