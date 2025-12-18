// Copyright 2024 Raido Pahtma. All rights reserved.

// A mist-cloud-comm AMQP (RabbitMQ) connection adapter.

package moteconnection

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

/*
Send to destination through any available gateway
	mist.FFFFFFFFFFFFFFFF.NODE

Send to destination through the specified gateway
	mist.GATEWAY.NODE

Send to cloud through the specified gateway
	cloud.GATEWAY.SERVER

Send to cloud through any gateway
	cloud.FFFFFFFFFFFFFFFF.SERVER
*/

// rabbitConnection - a grouping of amqp connection elements
type rabbitConnection struct {
	connection *amqp.Connection
	inchan     *amqp.Channel
	outchan    *amqp.Channel
}

// The MistCloudConnection object structure
type MistCloudConnection struct {
	BaseMoteConnection

	gateway EUI64

	name string

	rabbit string

	queueName string

	rbc *rabbitConnection

	exchangeName string

	connectedRabbit chan *rabbitConnection
	timeout         chan bool

	connectionClosed chan *amqp.Error
	incomingClosed   chan *amqp.Error
	outgoingClosed   chan *amqp.Error

	nodes map[EUI64]bool

	receiver chan Packet
	sender   chan Packet
}

// NewMistCloudConnection - Create a new MistCloudConnection
func NewMistCloudConnection(rabbiturl string) *MistCloudConnection {
	mcc := new(MistCloudConnection)
	mcc.InitLoggers()

	mcc.name = "moteconnection"
	mcc.gateway = 0 // 0 essentially disables comms, Configure must be called

	mcc.timeout = make(chan bool)
	mcc.connectedRabbit = make(chan *rabbitConnection)

	mcc.rabbit = rabbiturl
	mcc.exchangeName = "mistx"

	mcc.nodes = make(map[EUI64]bool)

	mcc.sender = make(chan Packet, 10)

	mcc.dispatchers = make(map[byte]Dispatcher)
	mcc.removeDispatcher = make(chan Dispatcher)
	mcc.addDispatcher = make(chan Dispatcher)

	mcc.watchdog = make(chan bool)
	mcc.closed = make(chan bool)
	mcc.close = make(chan bool)

	mcc.connected.Store(false)

	return mcc
}

func (mcc *MistCloudConnection) Configure(name string, gateway EUI64) {
	mcc.name = name
	mcc.gateway = gateway
}

func (mcc *MistCloudConnection) run() {
	var run_loops int = 0
	mcc.Debug.Printf("MistCloudConnection running\n")

	for {
		select {
		case <-mcc.close:
			mcc.shouldconnect.Store(false)

			mcc.connectlock.Lock()
			if mcc.rbc != nil {
				mcc.rbc.connection.Close()
			} else {
				mcc.connectlock.Unlock()
				mcc.closed <- true
				return
			}
			mcc.connectlock.Unlock()

		case rbbt := <-mcc.connectedRabbit:
			if rbbt != nil {
				if mcc.shouldconnect.Load() == false {
					rbbt.connection.Close()
					mcc.closed <- true
					return
				}
				mcc.Debug.Printf("connected")

				mcc.connectlock.Lock()
				mcc.rbc = rbbt
				mcc.connected.Store(true)

				mcc.connectionClosed = mcc.rbc.connection.NotifyClose(make(chan *amqp.Error, 1))
				mcc.incomingClosed = mcc.rbc.inchan.NotifyClose(make(chan *amqp.Error, 1))
				mcc.outgoingClosed = mcc.rbc.outchan.NotifyClose(make(chan *amqp.Error, 1))

				mcc.Debug.Printf("binding...")
				for eui64 := range mcc.nodes {
					err := mcc.bindNode(mcc.rbc.inchan, mcc.queueName, eui64)
					if err != nil {
						mcc.Warning.Printf("bind failure %s\n", err)
						mcc.rbc.connection.Close() // Start over
					}
				}

				go mcc.consume(mcc.rbc.inchan)

				mcc.connectlock.Unlock()
			} else {
				mcc.connectlock.Lock()
				mcc.rbc = nil
				mcc.connected.Store(false)

				if !mcc.autoconnect {
					mcc.shouldconnect.Store(false)
					mcc.closed <- true
					mcc.connectlock.Unlock()
					return
				}
				mcc.connectlock.Unlock()

				go mcc.wait(10 * time.Second)
			}

		case <-mcc.timeout:
			if mcc.shouldconnect.Load() == false {
				mcc.closed <- true
				return
			}
			go mcc.connectQueue(mcc.rabbit)

		case <-mcc.connectionClosed:
			if mcc.shouldconnect.Load() == false {
				mcc.Debug.Printf("connection closed")
			} else {
				mcc.Warning.Printf("connection closed")
			}

			mcc.connectionClosed = nil
			mcc.outgoingClosed = nil
			mcc.incomingClosed = nil

			mcc.connectlock.Lock()
			if mcc.rbc != nil {
				mcc.rbc = nil
				mcc.connected.Store(false)
			}
			mcc.connectlock.Unlock()

			if mcc.shouldconnect.Load() == false {
				mcc.closed <- true
				return
			}

			go mcc.connectQueue(mcc.rabbit)

		case <-mcc.incomingClosed:
			if mcc.shouldconnect.Load() == true {
				mcc.Warning.Printf("incoming channel closed")
			}

			mcc.incomingClosed = nil

			mcc.connectlock.Lock()
			if mcc.rbc != nil {
				mcc.rbc.connection.Close()
			}
			mcc.connectlock.Unlock()

		case <-mcc.outgoingClosed:
			if mcc.shouldconnect.Load() == true {
				mcc.Warning.Printf("outgoing channel closed")
			}

			mcc.outgoingClosed = nil

			mcc.connectlock.Lock()
			if mcc.rbc != nil {
				mcc.rbc.connection.Close()
			}
			mcc.connectlock.Unlock()

		case mp := <-mcc.sender:
			if mcc.rbc != nil {
				mcc.publishData(mcc.rbc.outchan, mp.(*MoteMistMessage))
			} else {
				mcc.Warning.Printf("message discarded")
			}

		case dispatcher := <-mcc.removeDispatcher:
			if mcc.dispatchers[dispatcher.Dispatch()] != dispatcher {
				panic("Asked to remove a dispatcher that is not registered!")
			}
			delete(mcc.dispatchers, dispatcher.Dispatch())

		case dispatcher := <-mcc.addDispatcher:
			mcc.dispatchers[dispatcher.Dispatch()] = dispatcher

		case <-time.After(20 * time.Minute): // Every 20 minutes
			run_loops++
			mcc.Debug.Printf("run loops %d", run_loops)
			mcc.connectlock.Lock()
			if mcc.rbc != nil {
				// TODO some periodic action with the AMQP connection?
				mcc.Debug.Printf("amqp is connected\n")
			}
			mcc.connectlock.Unlock()

		}
	}
}

// Connect to the cloud
func (mcc *MistCloudConnection) Connect() error {
	go mcc.run()
	mcc.timeout <- true
	for i := 0; i < 30; i++ {
		if mcc.connected.Load() == true {
			return nil
		}
		mcc.Debug.Printf("connecting...\n")
		time.Sleep(time.Second)
	}
	return errors.New("connection timed out")
}

func (mcc *MistCloudConnection) Autoconnect(period time.Duration) {
	mcc.connectlock.Lock()
	defer mcc.connectlock.Unlock()

	mcc.shouldconnect.Store(true)
	mcc.autoconnect = true
	mcc.period = period
	go mcc.run()
	mcc.timeout <- true
}

func (mcc *MistCloudConnection) Listen() error {
	return errors.New("cloud connection cannot be used for Listen")
}

func (mcc *MistCloudConnection) Send(msg Packet) error {
	mcc.sender <- msg
	return nil
}

func (mcc *MistCloudConnection) nodeKeys(eui EUI64) []string {
	if mcc.gateway == 0 {
		return []string{}
	} else if mcc.gateway == 0xFFFFFFFFFFFFFFFF {
		return []string{fmt.Sprintf("cloud.*.%s", eui)}
	} else {
		return []string{fmt.Sprintf("cloud.%s.%s", mcc.gateway, eui),
			fmt.Sprintf("cloud.FFFFFFFFFFFFFFFF.%s", eui)}
	}
}

func (mcc *MistCloudConnection) AddNode(eui EUI64) {
	mcc.connectlock.Lock()
	if _, ok := mcc.nodes[eui]; !ok {
		mcc.nodes[eui] = true
		if mcc.rbc != nil {
			err := mcc.bindNode(mcc.rbc.inchan, mcc.queueName, eui)
			if err != nil {
				mcc.Warning.Printf("bind failure %s\n", err)
				mcc.rbc.connection.Close() // Start over
			}
		}
	}
	mcc.connectlock.Unlock()
}

func (mcc *MistCloudConnection) RemoveNode(eui EUI64) {
	mcc.connectlock.Lock()

	delete(mcc.nodes, eui)

	for _, key := range mcc.nodeKeys(eui) {
		err := mcc.rbc.inchan.QueueUnbind(mcc.queueName, key, mcc.exchangeName, nil)
		if err != nil {
			mcc.Warning.Printf("unbind %s: %s\n", key, err)
			mcc.rbc.connection.Close()
			break
		}
	}

	mcc.connectlock.Unlock()
}

func (mdr *MistCloudConnection) RegisterReceiver(rchan chan Packet) {
	mdr.receiver = rchan
}

func (mcc *MistCloudConnection) publishData(ch *amqp.Channel, packet *MoteMistMessage) {

	mcc.AddNode(0x0015001500150015)
	//packet.Gateway = 0x5C0272FFFEA019A0 // 0xFFFFFFFFFFFFFFFF // TODO remove
	rkey := fmt.Sprintf("mist.%s.%s", packet.Gateway(), packet.Destination())

	data, err := packet.Serialize()
	if err != nil {
		mcc.Warning.Printf("%s", err)
		return
	}

	perr := ch.Publish(
		mcc.exchangeName, // exchange
		rkey,             // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{ContentType: "application/octet-stream", Body: data})
	if perr != nil {
		mcc.Warning.Printf("publish %s", err)
	} else {
		mcc.Debug.Printf("publish %s: [%s]%s", rkey, packet.Type(), hex.EncodeToString(packet.Payload))
	}
}

func (rrb *MistCloudConnection) wait(timeout time.Duration) {
	select {
	/*case <-rrb.done:
	return*/
	case <-time.After(timeout):
		rrb.timeout <- true
	}
}

func (mcc *MistCloudConnection) bindNode(chi *amqp.Channel, qname string, eui EUI64) error {
	for _, key := range mcc.nodeKeys(eui) {
		mcc.Debug.Printf("bind: %s\n", key)
		err := chi.QueueBind(
			qname,            // queue name
			key,              // routing key
			mcc.exchangeName, // exchange
			false,            // no-wait
			nil,              // arguments
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (mcc *MistCloudConnection) connectErrorCleanup(rbc *rabbitConnection, message string) error {
	mcc.Warning.Printf(message)
	if rbc.inchan != nil {
		rbc.inchan.Close()
	}
	if rbc.outchan != nil {
		rbc.outchan.Close()
	}
	if rbc.connection != nil {
		rbc.connection.Close()
	}
	mcc.connectedRabbit <- nil

	return nil
}

func (mcc *MistCloudConnection) connectQueue(url string) error {
	var rbc rabbitConnection
	var err error

	mcc.Debug.Printf("connecting...")
	rbc.connection, err = amqp.Dial(url)
	if err != nil {
		return mcc.connectErrorCleanup(&rbc, fmt.Sprintf("dial failure %s: %s", url, err))
	}

	mcc.Debug.Printf("creating channel...")
	rbc.inchan, err = rbc.connection.Channel()
	if err != nil {
		return mcc.connectErrorCleanup(&rbc, "channel failure")
	}

	mcc.Debug.Printf("checking exchange...")
	err = rbc.inchan.ExchangeDeclarePassive(
		mcc.exchangeName, // name
		"topic",          // type
		false,            // durable
		false,            // auto-delete
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil { // Channel gets closed if exchange does not exist!
		mcc.Warning.Printf("exchange does not exist\n")
		rbc.inchan, err = rbc.connection.Channel()
		if err != nil {
			return mcc.connectErrorCleanup(&rbc, "channel failure")
		}

		err = rbc.inchan.ExchangeDeclare(
			mcc.exchangeName, // name
			"topic",          // type
			false,            // durable
			false,            // auto-delete
			false,            // internal
			false,            // no-wait
			nil,              // arguments
		)
		if err != nil {
			return mcc.connectErrorCleanup(&rbc, fmt.Sprintf("exchange could not be created: %s\n", err))
		}
		mcc.Debug.Printf("exchange created")
	}

	mcc.Debug.Printf("creating publish channel...")
	rbc.outchan, err = rbc.connection.Channel()
	if err != nil {
		return mcc.connectErrorCleanup(&rbc, "channel failure")
	}

	mcc.queueName = fmt.Sprintf("%s-%s-%s", mcc.name, mcc.gateway, uuid.New())
	mcc.Debug.Printf("declaring queue... %s", mcc.queueName)
	_, err = rbc.inchan.QueueDeclare(
		mcc.queueName, // name
		false,         // durable
		true,          // auto-delete
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return mcc.connectErrorCleanup(&rbc, "queue failure")
	}

	mcc.connectedRabbit <- &rbc

	return nil
}

func (mcc *MistCloudConnection) consume(ch *amqp.Channel) {
	incoming, err := ch.Consume(
		mcc.queueName, // queue
		"",            // consumer
		true,          // auto-ack
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)

	if err != nil {
		mcc.Warning.Printf("consume failure %v", err)
		mcc.connectlock.Lock()
		if mcc.rbc != nil {
			mcc.rbc.connection.Close()
		}
		mcc.connectlock.Unlock()
		return
	}

	mcc.Debug.Printf("consuming...")
	for {
		msg, ok := <-incoming
		if !ok {
			if mcc.shouldconnect.Load() == true {
				mcc.Warning.Printf("incoming closed")
				mcc.connectlock.Lock()
				if mcc.rbc != nil {
					mcc.rbc.connection.Close()
				}
				mcc.connectlock.Unlock()
			}
			return
		}

		mcc.Debug.Printf("message received %+v", msg)

		if dispatcher, ok := mcc.dispatchers[0]; ok {
			err := dispatcher.Receive(msg.Body)
			if err != nil {
				mcc.Debug.Printf("Dispatcher error: %s", err)
			}
		} else {
			mcc.Debug.Printf("No dispatcher for %02X!\n", 0)
		}
	}
}
