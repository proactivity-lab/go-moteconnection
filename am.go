// Author  Raido Pahtma
// License MIT

package moteconnection

import (
	"encoding/hex"
	"fmt"
	"strconv"
)

// HexByte a byte object that should be converted to string with %02X
type HexByte byte

// HexString a bunch of bytes that should be converted to string with %02X
type HexString []byte

// AMAddr an address that should be represented as %04X
type AMAddr uint16

// AMID an ID that should be represented as %02X
type AMID uint8

// AMGroup a group ID that should be represented as %02X
type AMGroup uint8

// EUI64 address %016X
type EUI64 uint64

func (hxb HexByte) String() string {
	return fmt.Sprintf("%02X", byte(hxb))
}

// UnmarshalFlag unmarshals HexByte for the flags library
func (hxb *HexByte) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 8)
	*hxb = HexByte(data)
	return err
}

// MarshalFlag marshals HexByte for the flags library
func (hxb HexByte) MarshalFlag() (string, error) {
	return hxb.String(), nil
}

func (hxs HexString) String() string {
	return hex.EncodeToString(hxs)
}

// UnmarshalFlag unmarshals HexByte for the flags library
func (hxs *HexString) UnmarshalFlag(value string) error {
	data, err := hex.DecodeString(value)
	*hxs = data
	return err
}

// MarshalFlag marshals HexStrings for the flags library
func (hxs HexString) MarshalFlag() (string, error) {
	return hxs.String(), nil
}

func (addr AMAddr) String() string {
	return fmt.Sprintf("%04X", uint16(addr))
}

// UnmarshalFlag unmarshals AMAddr for the flags library
func (addr *AMAddr) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 16)
	*addr = AMAddr(data)
	return err
}

// MarshalFlag marshals AMAddr for the flags library
func (addr AMAddr) MarshalFlag() (string, error) {
	return addr.String(), nil
}

func (aid AMID) String() string {
	return fmt.Sprintf("%02X", uint8(aid))
}

// UnmarshalFlag unmarshals AMID for the flags library
func (aid *AMID) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 8)
	*aid = AMID(data)
	return err
}

// MarshalFlag marshals AMID for the flags library
func (aid AMID) MarshalFlag() (string, error) {
	return aid.String(), nil
}

func (grp AMGroup) String() string {
	return fmt.Sprintf("%02X", uint8(grp))
}

// UnmarshalFlag unmarshals AMGroup for the flags library
func (grp *AMGroup) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 8)
	*grp = AMGroup(data)
	return err
}

// MarshalFlag marshals AMGroup for the flags library
func (grp AMGroup) MarshalFlag() (string, error) {
	return grp.String(), nil
}

func (eui64 EUI64) String() string {
	return fmt.Sprintf("%016X", uint64(eui64))
}

// UnmarshalFlag unmarshals EUI64 for the flags library
func (eui64 *EUI64) UnmarshalFlag(value string) error {
	data, err := strconv.ParseUint(value, 16, 64)
	*eui64 = EUI64(data)
	return err
}

// MarshalFlag marshals EUI64 for the flags library
func (eui64 EUI64) MarshalFlag() (string, error) {
	return eui64.String(), nil
}
