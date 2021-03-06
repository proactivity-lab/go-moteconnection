// Author  Raido Pahtma
// License MIT

package moteconnection

import "fmt"
import "time"
import "errors"

type PacketDispatcher struct {
	factory  PacketFactory
	receiver chan Packet
}

type MessageDispatcher struct {
	factory   *Message
	receivers map[AMID]chan Packet
	snooper   chan Packet
}

var _ Dispatcher = (*PacketDispatcher)(nil)
var _ Dispatcher = (*MessageDispatcher)(nil)

func (self *PacketDispatcher) Receive(msg []byte) error {
	p := self.factory.NewPacket()
	err := p.Deserialize(msg)
	if err == nil {
	DeliverReceiver:
		for self.receiver != nil {
			select {
			case self.receiver <- p:
				break DeliverReceiver
			case <-time.After(50 * time.Millisecond):
			}
		}
	} else {
		return errors.New(fmt.Sprintf("Deserialize error: %s", err))
	}
	return nil
}

func (self *MessageDispatcher) Receive(msg []byte) error {
	p := self.factory.NewPacket().(*Message)
	err := p.Deserialize(msg)
	if err == nil {
	DeliverReceiver:
		for rcvr, ok := self.receivers[p.Type()]; ok; rcvr, ok = self.receivers[p.Type()] {
			select {
			case rcvr <- p:
				break DeliverReceiver
			case <-time.After(50 * time.Millisecond):
			}
		}
	DeliverSnooper:
		for self.snooper != nil {
			select {
			case self.snooper <- p:
				break DeliverSnooper
			case <-time.After(50 * time.Millisecond):
			}
		}
	} else {
		return errors.New(fmt.Sprintf("Deserialize error: %s", err))
	}
	return nil
}

func (self *PacketDispatcher) Dispatch() byte {
	return self.factory.Dispatch()
}

func (self *MessageDispatcher) Dispatch() byte {
	return self.factory.Dispatch()
}

func (self *PacketDispatcher) NewPacket() Packet {
	return self.factory.NewPacket()
}

func (self *MessageDispatcher) NewPacket() Packet {
	return self.factory.NewPacket()
}

func (self *MessageDispatcher) NewMessage() *Message {
	return self.factory.NewPacket().(*Message)
}

func (self *PacketDispatcher) RegisterReceiver(receiver chan Packet) error {
	self.receiver = receiver
	return nil
}

func (self *MessageDispatcher) RegisterMessageSnooper(receiver chan Packet) error {
	self.snooper = receiver
	return nil
}

func (self *MessageDispatcher) RegisterMessageReceiver(amid AMID, receiver chan Packet) error {
	self.receivers[amid] = receiver
	return nil
}

func (self *MessageDispatcher) DeregisterMessageReceiver(amid AMID) error {
	delete(self.receivers, amid)
	return nil
}

func NewPacketDispatcher(packetfactory PacketFactory) *PacketDispatcher {
	d := new(PacketDispatcher)
	d.factory = packetfactory
	return d
}

func NewMessageDispatcher(packetfactory *Message) *MessageDispatcher {
	d := new(MessageDispatcher)
	d.factory = packetfactory
	d.receivers = make(map[AMID]chan Packet)
	return d
}
