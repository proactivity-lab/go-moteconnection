// Author  Raido Pahtma
// License MIT

package moteconnection

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/proactivity-lab/go-loggers"
)

// Packet interface
type Packet interface {
	Dispatch() byte
	Serialize() ([]byte, error)
	Deserialize([]byte) error
	SetPayload([]byte) error
	GetPayload() []byte
}

// PacketFactory interface
type PacketFactory interface {
	Dispatch() byte
	NewPacket() Packet
}

// Dispatcher interface
type Dispatcher interface {
	Dispatch() byte
	Receive([]byte) error
	NewPacket() Packet
}

// MoteConnection interface
type MoteConnection interface {
	loggers.DIWElog
	Listen() error
	Connect() error
	Autoconnect(period time.Duration)
	Send(msg Packet) error
	AddDispatcher(dispatcher Dispatcher) error
	RemoveDispatcher(dispatch Dispatcher) error
	Connected() bool
	Disconnect()
}

// BaseMoteConnection structure
type BaseMoteConnection struct {
	loggers.DIWEloggers

	connectlock   sync.Mutex
	shouldconnect atomic.Value
	autoconnect   bool
	period        time.Duration

	connected atomic.Value
	outgoing  chan []byte
	incoming  chan []byte

	dispatchers      map[byte]Dispatcher
	removeDispatcher chan Dispatcher
	addDispatcher    chan Dispatcher

	watchdog chan bool
	closed   chan bool
	close    chan bool
}

func (bmc *BaseMoteConnection) notifyWatchdog(outcome bool) {
	select {
	case bmc.watchdog <- outcome:
	case <-time.After(50 * time.Millisecond): // Because notification could happen before the watchdog has started
		bmc.Debug.Printf("No watchdog?\n")
	}
}

// Send sends a packet
func (bmc *BaseMoteConnection) Send(msg Packet) error {
	serialized, err := msg.Serialize()
	if err != nil {
		return err
	}
	select {
	case bmc.outgoing <- serialized:
		return nil
	case <-time.After(50 * time.Millisecond): // Because the run goroutine might be doing something other than reading bmc.outgoing at this very moment
		return errors.New("not connected")
	}
}

// AddDispatcher adds a dispatcher to the connection
func (bmc *BaseMoteConnection) AddDispatcher(dispatcher Dispatcher) error {
	bmc.connectlock.Lock()
	defer bmc.connectlock.Unlock()
	if bmc.connected.Load().(bool) {
		bmc.addDispatcher <- dispatcher
	} else {
		bmc.dispatchers[dispatcher.Dispatch()] = dispatcher
	}
	return nil
}

// RemoveDispatcher removes a dispatcher from the connection
func (bmc *BaseMoteConnection) RemoveDispatcher(dispatcher Dispatcher) error {
	bmc.connectlock.Lock()
	defer bmc.connectlock.Unlock()
	if bmc.connected.Load().(bool) {
		bmc.removeDispatcher <- dispatcher
	} else {
		delete(bmc.dispatchers, dispatcher.Dispatch())
	}
	return nil
}

// Connected checks if the connection is connected
func (bmc *BaseMoteConnection) Connected() bool {
	return bmc.connected.Load().(bool)
}

// Disconnect disconnects the connection or stops a server
func (bmc *BaseMoteConnection) Disconnect() {
	bmc.connectlock.Lock()
	bmc.shouldconnect.Store(false)
	bmc.autoconnect = false
	bmc.notifyWatchdog(false) // Because of autoconnect, watchdog may be active
	bmc.connectlock.Unlock()

	close(bmc.close)

	// Wait for connections to close
	for {
		if bmc.Connected() {
			time.Sleep(time.Millisecond)
		} else {
			return
		}
	}
}
