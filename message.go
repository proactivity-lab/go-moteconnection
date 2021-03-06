// Author  Raido Pahtma
// License MIT

package moteconnection

import "encoding/binary"
import "bytes"
import "errors"
import "fmt"

type Message struct {
	dispatch    byte
	destination AMAddr
	source      AMAddr
	sourceSet   bool
	group       AMGroup
	groupSet    bool
	ptype       AMID
	Payload     []byte
	Footer      []byte

	LQI  uint8 // LQI is set if footer is exactly 2 bytes long
	RSSI int8  // RSSI is set if footer is exactly 2 bytes long

	defaultSource AMAddr
	defaultGroup  AMGroup
}

type LinkQualitySignalStrengthFooter struct {
	lqi  uint8
	rssi int8
}

var _ Packet = (*Message)(nil)
var _ PacketFactory = (*Message)(nil)

func NewMessage(defaultGroup AMGroup, defaultSource AMAddr) *Message {
	msg := new(Message)
	msg.dispatch = 0
	msg.defaultGroup = defaultGroup
	msg.defaultSource = defaultSource
	return msg
}

// Message also serves as a factory.
func (self *Message) NewPacket() Packet {
	msg := new(Message)
	msg.dispatch = self.dispatch
	msg.defaultGroup = self.defaultGroup
	msg.defaultSource = self.defaultSource
	return msg
}

func (self *Message) Dispatch() byte {
	return self.dispatch
}

func (self *Message) SetPayload(payload []byte) error {
	self.Payload = payload
	return nil
}

func (self *Message) GetPayload() []byte {
	return self.Payload
}

func (self *Message) SetDispatch(dispatch byte) {
	self.dispatch = dispatch
}

func (self *Message) Type() AMID {
	return self.ptype
}

func (self *Message) SetType(ptype AMID) {
	self.ptype = ptype
}

func (self *Message) Group() AMGroup {
	if self.groupSet {
		return self.group
	}
	return self.defaultGroup
}

func (self *Message) SetGroup(group AMGroup) {
	self.groupSet = true
	self.group = group
}

func (self *Message) Destination() AMAddr {
	return self.destination
}

func (self *Message) SetDestination(destination AMAddr) {
	self.destination = destination
}

func (self *Message) Source() AMAddr {
	if self.sourceSet {
		return self.source
	}
	return self.defaultSource
}

func (self *Message) SetSource(source AMAddr) {
	self.sourceSet = true
	self.source = source
}

func (self *Message) String() string {
	s := fmt.Sprintf("{%s}%s->%s[%s]%3d: %X", self.Group(), self.Source(), self.destination, self.ptype, len(self.Payload), self.Payload)
	if self.LQI != 0 && self.RSSI != 0 {
		s = fmt.Sprintf("%s %02X:%3d", s, self.LQI, self.RSSI)
	}
	return s
}

func (self *Message) Serialize() ([]byte, error) {
	var err error
	buf := new(bytes.Buffer)

	if len(self.Payload) > 255-8 {
		return nil, errors.New(fmt.Sprintf("Message payload too long(%d)", len(self.Payload)))
	}

	err = binary.Write(buf, binary.BigEndian, self.dispatch)
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, self.Destination())
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, self.Source())
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, uint8(len(self.Payload)))
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, self.Group())
	if err != nil {
		panic(err)
	}

	err = binary.Write(buf, binary.BigEndian, self.Type())
	if err != nil {
		panic(err)
	}

	_, err = buf.Write(self.Payload)
	if err != nil {
		panic(err)
	}

	if len(self.Footer) > 0 {
		_, err = buf.Write(self.Footer)
		if err != nil {
			panic(err)
		}
	}

	return buf.Bytes(), nil
}

func (self *Message) Deserialize(data []byte) error {
	var err error

	var dispatch uint8
	var destination AMAddr
	var source AMAddr
	var length uint8
	var group AMGroup
	var ptype AMID

	buf := bytes.NewReader(data)

	err = binary.Read(buf, binary.BigEndian, &dispatch)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &destination)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &source)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &length)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &group)
	if err != nil {
		return err
	}

	err = binary.Read(buf, binary.BigEndian, &ptype)
	if err != nil {
		return err
	}

	if (uint8)(buf.Len()) < length {
		return errors.New(fmt.Sprintf("Payload too short - header=%d, actual=%d", length, buf.Len()))
	}

	payload := make([]byte, length)
	_, err = buf.Read(payload)
	if err != nil {
		return err
	}

	self.dispatch = dispatch
	self.SetDestination(destination)
	self.SetSource(source)
	self.SetGroup(group)
	self.SetType(ptype)
	self.Payload = payload

	if buf.Len() > 0 {
		footer := make([]byte, buf.Len())
		_, err = buf.Read(footer)
		if err != nil {
			return err
		}
		self.Footer = footer

		if len(self.Footer) == 2 {
			footerbuf := bytes.NewReader(self.Footer)

			err = binary.Read(footerbuf, binary.BigEndian, &self.LQI)
			if err != nil {
				return err
			}

			err = binary.Read(footerbuf, binary.BigEndian, &self.RSSI)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
