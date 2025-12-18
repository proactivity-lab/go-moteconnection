// Author  Raido Pahtma
// License MIT

package moteconnection

import (
	"fmt"
	"time"
)

// PacketDispatcher structure
type PacketDispatcher struct {
	factory  PacketFactory
	receiver chan Packet
}

type MessageDispatcherMessage interface {
	Packet
	Type() AMID;
	NewPacket() Packet;
}

// MessageDispatcher structure
type MessageDispatcher struct {
	factory   MessageDispatcherMessage
	receivers map[AMID]chan Packet
	snooper   chan Packet
}

// MessageDispatcher structure
type MistMessageDispatcher struct {
	factory   MessageDispatcherMessage
	receivers map[AMID]chan Packet
	snooper   chan Packet
}

var _ Dispatcher = (*PacketDispatcher)(nil)
var _ Dispatcher = (*MessageDispatcher)(nil)

// Receive distributes a packet to receivers
func (pd *PacketDispatcher) Receive(msg []byte) error {
	p := pd.factory.NewPacket()
	err := p.Deserialize(msg)
	if err == nil {
	DeliverReceiver:
		for pd.receiver != nil {
			select {
			case pd.receiver <- p:
				break DeliverReceiver
			case <-time.After(50 * time.Millisecond):
			}
		}
	} else {
		return fmt.Errorf("Deserialize error: %s", err)
	}
	return nil
}

// Receive distributes a packet to receivers and snoopers
func (md *MessageDispatcher) Receive(msg []byte) error {
	p := md.factory.NewPacket().(MessageDispatcherMessage)
	err := p.Deserialize(msg)
	if err == nil {
	DeliverReceiver:
		for rcvr, ok := md.receivers[p.Type()]; ok; rcvr, ok = md.receivers[p.Type()] {
			select {
			case rcvr <- p:
				break DeliverReceiver
			case <-time.After(50 * time.Millisecond):
			}
		}
	DeliverSnooper:
		for md.snooper != nil {
			select {
			case md.snooper <- p:
				break DeliverSnooper
			case <-time.After(50 * time.Millisecond):
			}
		}
	} else {
		return fmt.Errorf("Deserialize error: %s", err)
	}
	return nil
}

// Dispatch returns the dispatch ID of the dispatcher instance
func (pd *PacketDispatcher) Dispatch() byte {
	return pd.factory.Dispatch()
}

// Dispatch returns the dispatch ID of the dispatcher instance
func (md *MessageDispatcher) Dispatch() byte {
	return md.factory.Dispatch()
}

// Dispatch returns the dispatch ID of the dispatcher instance
func (md *MistMessageDispatcher) Dispatch() byte {
	return md.factory.Dispatch()
}

// NewPacket initializes a new packet for use with this dispatcher
func (pd *PacketDispatcher) NewPacket() Packet {
	return pd.factory.NewPacket()
}

// NewPacket initializes a new packet for use with this dispatcher
func (md *MessageDispatcher) NewPacket() Packet {
	return md.factory.NewPacket()
}

// NewPacket initializes a new packet for use with this dispatcher
func (md *MistMessageDispatcher) NewPacket() Packet {
	return md.factory.NewPacket()
}

// NewMessage initializes a new message for use with this dispatcher
func (md *MessageDispatcher) NewMessage() *Message {
	return md.factory.NewPacket().(*Message)
}

// RegisterReceiver registers a receiver
func (pd *PacketDispatcher) RegisterReceiver(receiver chan Packet) error {
	pd.receiver = receiver
	return nil
}

// RegisterMessageSnooper registers a snooper
func (md *MessageDispatcher) RegisterMessageSnooper(receiver chan Packet) error {
	md.snooper = receiver
	return nil
}

// RegisterMessageReceiver registers a receiver
func (md *MessageDispatcher) RegisterMessageReceiver(amid AMID, receiver chan Packet) error {
	md.receivers[amid] = receiver
	return nil
}

// DeregisterMessageReceiver removes a receiver
func (md *MessageDispatcher) DeregisterMessageReceiver(amid AMID) error {
	delete(md.receivers, amid)
	return nil
}

// NewPacketDispatcher creates a new dispatcher
func NewPacketDispatcher(packetfactory PacketFactory) *PacketDispatcher {
	d := new(PacketDispatcher)
	d.factory = packetfactory
	return d
}

// NewMessageDispatcher creates a new dispatcher
func NewMessageDispatcher(packetfactory MessageDispatcherMessage) *MessageDispatcher {
	d := new(MessageDispatcher)
	d.factory = packetfactory
	d.receivers = make(map[AMID]chan Packet)
	return d
}
