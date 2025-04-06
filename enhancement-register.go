package modbus

import (
	"encoding/json"
	"fmt"
)

// DeviceRegister represents a Modbus register with metadata
type DeviceRegister struct {
	Tag       string  `json:"tag"`
	Alias     string  `json:"alias"`
	Function  int     `json:"function"`
	SlaverId  byte    `json:"slaverId"`
	Address   uint16  `json:"address"`
	Frequency int64   `json:"frequency"`
	Quantity  uint16  `json:"quantity"`
	DataType  string  `json:"dataType"`  // uint16, int16, uint32, int32, float32, float64
	DataOrder string  `json:"dataOrder"` // ABCD, DCBA, CDAB, DCBA
	Weight    float64 `json:"weight"`
	Value     [4]byte `json:"value"`
}

// Encode Bytes
func (r DeviceRegister) Encode() []byte {
	return r.Value[:]
}

// Decode Bytes
func (r *DeviceRegister) Decode(data []byte) {
	copy(r.Value[:], data)
}

// To string
func (r DeviceRegister) String() string {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return string(jsonData)
}
