// Author  Raido Pahtma
// License MIT

package moteconnection

import (
	"time"
	"fmt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type MoteMistMessage struct {
	dispatch    byte
	gateway     EUI64
	destination EUI64
	source      EUI64
	group       AMGroup
	amid        AMID
	Payload     []byte
	Footer      []byte

	LQI  uint8
	RSSI int8

	defaultSource EUI64
	defaultGroup  AMGroup
	defaultGateway EUI64

	Timestamp time.Time
}

var _ Packet = (*MoteMistMessage)(nil)
var _ PacketFactory = (*MoteMistMessage)(nil)

func NewMoteMistMessage(defaultGroup AMGroup, defaultGateway EUI64, defaultSource EUI64) *MoteMistMessage {
	msg := new(MoteMistMessage)
	msg.dispatch = 0
	msg.defaultGroup = defaultGroup
	msg.defaultSource = defaultSource
	msg.defaultGateway = defaultGateway
	return msg
}

// Implement Packet
func (mmm *MoteMistMessage) NewPacket() Packet {
	msg := new(MoteMistMessage)
	msg.dispatch = mmm.dispatch
	msg.defaultGroup = mmm.defaultGroup
	msg.defaultSource = mmm.defaultSource
	msg.defaultGateway = mmm.defaultGateway
	return msg
}

func (mmm *MoteMistMessage) Serialize() ([]byte, error) {
	var mm MistMessage
	mm.Timestamp = timestamppb.New(mmm.Timestamp)
	mm.Destination = uint64(mmm.Destination())
	mm.Source = uint64(mmm.Source())
	mm.Gateway = uint64(mmm.Gateway())
	mm.Group = 0x3F00 | uint32(mmm.Group())
	mm.Amid = uint32(mmm.Type())
	mm.Channel = 0 // uint32(mmm.Channel)
	mm.Rssi = int32(mmm.RSSI)
	mm.Lqi = uint32(mmm.LQI)
	mm.Payload = mmm.Payload

	data, err := proto.Marshal(&mm)
	if err != nil {
		fmt.Errorf("Failed to encode message: %s", err)
	}
	return data, nil
}

func (mmm *MoteMistMessage) Deserialize(data []byte) error {
	mm := &MistMessage{}
	if err := proto.Unmarshal(data, mm); err == nil {
		mmm.SetSource(EUI64(mm.Source))
		mmm.SetDestination(EUI64(mm.Destination))
		mmm.SetGateway(EUI64(mm.Gateway))
		mmm.SetType(AMID(mm.Amid))
		mmm.SetGroup(AMGroup(mm.Group))
		mmm.Timestamp = mm.Timestamp.AsTime()
		mmm.Payload = mm.Payload
	} else {
		return fmt.Errorf("Failed to parse message: %s\n", err)
	}
	return nil
}

func (mmm *MoteMistMessage) Dispatch() byte {
	return mmm.dispatch
}

func (mmm *MoteMistMessage) GetPayload() []byte {
	return mmm.Payload
}

func (mmm *MoteMistMessage) SetPayload(payload []byte) error {
	mmm.Payload = payload
	return nil
}

func (mmm *MoteMistMessage) SetDispatch(dispatch byte) {
	mmm.dispatch = dispatch
}

// Implement Message

func (mmm *MoteMistMessage) Type() AMID {
	return mmm.amid
}

func (mmm *MoteMistMessage) SetType(amid AMID) {
	mmm.amid = amid
}

func (mmm *MoteMistMessage) Destination() EUI64 {
	return mmm.destination
}

func (mmm *MoteMistMessage) SetDestination(destination EUI64) {
	mmm.destination = destination
}

func (mmm *MoteMistMessage) Source() EUI64 {
	if mmm.source == 0 {
		return mmm.defaultSource
	}
	return mmm.source
}

func (mmm *MoteMistMessage) SetSource(source EUI64) {
	mmm.source = source
}

func (mmm *MoteMistMessage) Gateway() EUI64 {
	if mmm.gateway == 0 {
		return mmm.defaultGateway
	}
	return mmm.gateway
}

func (mmm *MoteMistMessage) SetGateway(gateway EUI64) {
	mmm.gateway = gateway
}

func (mmm *MoteMistMessage) Group() AMGroup {
	if mmm.group == 0 {
		return mmm.defaultGroup
	}
	return mmm.group
}

func (mmm *MoteMistMessage) SetGroup(group AMGroup) {
	mmm.group = group
}
