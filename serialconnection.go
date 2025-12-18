// Author  Raido Pahtma
// License MIT

// Serial Smart Dust Mote connection.
package moteconnection

import (
	"bytes"
	"encoding/binary"
	"errors"
	"syscall"
	"time"

	"github.com/joaojeronimo/go-crc16"
	"go.bug.st/serial.v1"
)

const IncomingBufferSize = 10

type SerialConnection struct {
	BaseMoteConnection

	Port string
	Baud int

	acks chan byte

	seqnum byte

	conn serial.Port
}

func (self *SerialConnection) runErrorHandler(err error) error {
	// if err != Some known common error {
	self.Debug.Printf("SCERR: %s\n", err)
	// }
	return err
}

func (self *SerialConnection) connectErrorHandler(conn serial.Port, err error) error {
	if conn != nil {
		conn.Close()
	}
	if self.autoconnect {
		go self.connect(self.period)
	}
	return self.runErrorHandler(err)
}

func (self *SerialConnection) Listen() error {
	return errors.New("Serial connection cannot be used for Listen")
}

func (self *SerialConnection) Connect() error {
	self.connectlock.Lock()

	self.shouldconnect.Store(true)
	self.autoconnect = false

	self.connectlock.Unlock()

	return self.connect(0)
}

func (self *SerialConnection) Autoconnect(period time.Duration) {
	self.connectlock.Lock()
	defer self.connectlock.Unlock()

	self.shouldconnect.Store(true)
	self.autoconnect = true
	self.period = period
	go self.connect(0)
}

func (self *SerialConnection) connectWatchdog(timeout time.Duration, conn serial.Port) {
	select {
	case v := <-self.watchdog:
		if v {
			self.Debug.Printf("Watchdog stop.\n")
			return
		} else {
			self.Debug.Printf("Watchdog interrupt.\n")
			conn.Close()
		}
	case <-time.After(timeout):
		self.Debug.Printf("Timeout.\n")
		conn.Close()
	}
}

func (self *SerialConnection) connect(delay time.Duration) error {
	time.Sleep(delay)

	self.connectlock.Lock()
	defer self.connectlock.Unlock()

	if self.connected.Load().(bool) {
		return errors.New("Already connected.")
	}

	if self.shouldconnect.Load().(bool) == false {
		return errors.New("Connect interrupted.")
	}

	mode := &serial.Mode{BaudRate: self.Baud, Parity: serial.EvenParity, DataBits: 8, StopBits: serial.OneStopBit}

	port, err := serial.Open(self.Port, mode)

	port.SetDTR(false)

	if err == nil {
		self.notifyWatchdog(true)
		self.connected.Store(true)

		self.conn = port

		go self.read()
		go self.run()

		return nil
	}

	return self.connectErrorHandler(nil, err)
}

func (self *SerialConnection) read() {
	var seqnum int = -1 // seqnum is byte, but use int to have an initial value that is impossible
	escape := false
	buf := new(bytes.Buffer)
	for {
		p := make([]byte, 16)
		length, err := self.conn.Read(p)
		if err == nil {
			if length > 0 {
				p = p[:length]
				self.Debug.Printf("i %d %x\n", len(p), p)
				for _, b := range p {
					if escape {
						b = b ^ 0x20
						escape = false
					} else if b == 0x7D {
						escape = true
						continue
					} else if b == 0x7E { // Must always end with 0x7E, does not need to start with one
						escape = false
						if buf.Len() >= 4 {
							pckt := buf.Bytes()
							ccrc := crc16.Crc16(pckt[:len(pckt)-2])
							pcrc := uint16(pckt[len(pckt)-1])<<8 + uint16(pckt[len(pckt)-2])
							self.Debug.Printf("%d %x %04x\n", len(pckt), pckt, pcrc)
							if pcrc == ccrc {
								switch pckt[0] {
								case 0x45: // noackpacket
									self.incoming <- pckt[1 : len(pckt)-2]
								case 0x44: // ackpacket
									if seqnum != int(pckt[1]) {
										self.incoming <- pckt[2 : len(pckt)-2]
										seqnum = int(pckt[1])
									}
									// TODO send ack
								case 0x43: // ack
									self.Debug.Printf("ack %02x", pckt[1])
									self.acks <- pckt[1]
								}
							} else {
								self.Warning.Printf("CRC mismatch %04x != %04x\n", ccrc, pcrc)
							}
						} else {
							// not enough data received
						}
						buf = new(bytes.Buffer)
						continue
					}
					err = binary.Write(buf, binary.BigEndian, b)
				}
			} else {
				self.runErrorHandler(errors.New("Received 0 bytes"))
				self.closed <- true
				break
			}
		} else if err == syscall.EINTR {
			//self.Debug.Printf("EINTR\n")
			continue
		} else {
			self.runErrorHandler(err)
			self.closed <- true
			break
		}
	}
}

func (self *SerialConnection) send(data []byte) error {
	self.seqnum++
	pckt := []byte{0x44, self.seqnum}
	//pckt[0] = 0x44
	//pckt[1] = self.seqnum
	pckt = append(pckt, data...)
	crc := crc16.Crc16(pckt)
	pckt = append(pckt, byte(crc&0xff))
	pckt = append(pckt, byte(crc>>8))
	// self.Debug.Printf("snd(%d): %X\n", len(pckt), pckt)

	raw := []byte{0x7e}
	for _, b := range pckt {
		if b == 0x7e || b == 0x7d {
			raw = append(raw, byte(0x7d), byte(b^0x20))
		} else {
			raw = append(raw, b)
		}
	}
	raw = append(raw, byte(0x7e))
	self.Debug.Printf("o %d: %x\n", len(raw), raw)

	_, err := self.conn.Write(raw)
	return err
}

func (self *SerialConnection) run() {
	for {
		select {
		case msg := <-self.incoming:
			self.Debug.Printf("RCV(%d): %X\n", len(msg), msg)
			if len(msg) > 0 {
				if dispatcher, ok := self.dispatchers[msg[0]]; ok {
					err := dispatcher.Receive(msg)
					if err != nil {
						self.Debug.Printf("Dispatcher error: %s", err)
					}
				} else {
					self.Debug.Printf("No dispatcher for %02X!\n", msg[0])
				}
			}
		case ack := <-self.acks: // discard the ack here, it is too late
			self.Debug.Printf("Late ack %02x\n", ack)
		case msg := <-self.outgoing:
			self.Debug.Printf("SND(%d): %X\n", len(msg), msg)

			self.send(msg)

			timeout := time.After(100 * time.Millisecond)
			done := false
			for done == false {
				select {
				case ack := <-self.acks:
					if ack == self.seqnum { // delivered successfully
						self.Debug.Printf("ACK %02x", ack)
						// TODO signal some sendDone event?
						done = true
					} else {
						self.Debug.Printf("Unexpected ack %02x\n", ack)
					}
				case <-timeout:
					self.Warning.Printf("Ack %02x not received!\n", self.seqnum)
					done = true
				}
			}
		case <-self.close:
			self.Debug.Printf("Closing.\n")
			self.close = nil
			self.conn.Close()
		case <-self.closed:
			self.Debug.Printf("Connection closed.\n")
			self.connectlock.Lock()
			defer self.connectlock.Unlock()

			self.connected.Store(false)
			self.conn.Close()
			if self.autoconnect {
				go self.connect(self.period)
			}
			return
		case dispatcher := <-self.removeDispatcher:
			if self.dispatchers[dispatcher.Dispatch()] != dispatcher {
				panic("Asked to remove a dispatcher that is not registered!")
			}
			delete(self.dispatchers, dispatcher.Dispatch())
		case dispatcher := <-self.addDispatcher:
			self.dispatchers[dispatcher.Dispatch()] = dispatcher
		}
	}
}

func ListSerialPorts() ([]string, error) {
	ports, err := serial.GetPortsList()
	return ports, err
}

func NewSerialConnection(port string, baud int) *SerialConnection {
	sc := new(SerialConnection)
	sc.InitLoggers()
	sc.Port = port
	sc.Baud = baud
	sc.dispatchers = make(map[byte]Dispatcher)
	sc.removeDispatcher = make(chan Dispatcher)
	sc.addDispatcher = make(chan Dispatcher)
	sc.outgoing = make(chan []byte)
	sc.incoming = make(chan []byte, IncomingBufferSize)
	sc.acks = make(chan byte)
	sc.watchdog = make(chan bool)
	sc.closed = make(chan bool)
	sc.close = make(chan bool)
	sc.connected.Store(false)
	return sc
}
