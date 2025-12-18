// A SerialForwarder implementation using the Go MoteConnection library.
// Compared to the original variants:
//  * it correctly increments sequence numbers
//  * it will recover from a UART/USB disconnect
//  * can act as a server or client on both ends
//  * can also forward a TCP source connection

// @author Raido Pahtma
// @license MIT

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/proactivity-lab/go-moteconnection"
)

// ApplicationVersionMajor -
const ApplicationVersionMajor = 0

// ApplicationVersionMinor -
const ApplicationVersionMinor = 0

// ApplicationVersionPatch -
const ApplicationVersionPatch = 1

// ApplicationBuildDate -
var ApplicationBuildDate string

// ApplicationBuildDistro -
var ApplicationBuildDistro string

func main() {

	var opts struct {
		Positional struct {
			ServerConnectionString string `description:"Connectionstring sf/udp@HOST:PORT" default:"sf@0.0.0.0:9002"`
			ClientConnectionString string `description:"Connectionstring sf@HOST:PORT or serial@PORT:BAUD" default:"serial@/dev/ttyUSB0:115200"`
		} `positional-args:"yes"`

		Reconnect uint `long:"reconnect" default:"10" description:"Reconnect period, seconds"`

		ClientClient bool `long:"client-client" description:"Dial on both ends"`
		ServerServer bool `long:"server-server" description:"Listen on both ends"`

		Debug       []bool `short:"D" long:"debug" description:"Debug mode, print raw packets"`
		ShowVersion func() `short:"V" long:"version" description:"Show application version"`
	}

	opts.ShowVersion = func() {
		if ApplicationBuildDate == "" {
			ApplicationBuildDate = "YYYY-mm-dd_HH:MM:SS"
		}
		if ApplicationBuildDistro == "" {
			ApplicationBuildDistro = "unknown"
		}
		fmt.Printf("sforwarder %d.%d.%d (%s %s)\n",
			ApplicationVersionMajor, ApplicationVersionMinor, ApplicationVersionPatch,
			ApplicationBuildDate, ApplicationBuildDistro)
		os.Exit(0)
	}

	_, err := flags.Parse(&opts)
	if err != nil {
		flagserr := err.(*flags.Error)
		if flagserr.Type != flags.ErrHelp {
			if len(opts.Debug) > 0 {
				fmt.Printf("Argument parser error: %s\n", err)
			}
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Applying both would effectively swap the ends, which makes no sense
	if opts.ClientClient && opts.ServerServer {
		fmt.Printf("ERROR: client-client, server-server or neither of them, NOT both!\n")
		os.Exit(1)
	}

	sconn, scs, err := moteconnection.CreateConnection(opts.Positional.ServerConnectionString)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}

	cconn, ccs, err := moteconnection.CreateConnection(opts.Positional.ClientConnectionString)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}

	// Configure logging
	logformat := log.Ldate | log.Ltime | log.Lmicroseconds
	var logger *log.Logger
	if len(opts.Debug) > 0 {
		if len(opts.Debug) > 1 {
			logformat = logformat | log.Lshortfile
		}
		logger = log.New(os.Stdout, "INFO:  ", logformat)
		sconn.SetDebugLogger(log.New(os.Stdout, "DEBUG: ", logformat))
		sconn.SetInfoLogger(logger)
		cconn.SetDebugLogger(log.New(os.Stdout, "DEBUG: ", logformat))
		cconn.SetInfoLogger(logger)
	} else {
		logger = log.New(os.Stdout, "", logformat)
	}
	sconn.SetWarningLogger(log.New(os.Stdout, "WARN:  ", logformat))
	sconn.SetErrorLogger(log.New(os.Stdout, "ERROR: ", logformat))
	cconn.SetWarningLogger(log.New(os.Stdout, "WARN:  ", logformat))
	cconn.SetErrorLogger(log.New(os.Stdout, "ERROR: ", logformat))

	// Set up dispatchers for all possible dispatch IDs on both ends
	var cdsps []moteconnection.Dispatcher
	creceive := make(chan moteconnection.Packet)
	for i := 0; i <= 255; i++ {
		dsp := moteconnection.NewPacketDispatcher(moteconnection.NewRawPacket(byte(i)))
		dsp.RegisterReceiver(creceive)
		cconn.AddDispatcher(dsp)
		cdsps = append(cdsps, dsp)
	}

	var sdsps []moteconnection.Dispatcher
	sreceive := make(chan moteconnection.Packet)
	for i := 0; i <= 255; i++ {
		dsp := moteconnection.NewPacketDispatcher(moteconnection.NewRawPacket(byte(i)))
		dsp.RegisterReceiver(sreceive)
		sconn.AddDispatcher(dsp)
		sdsps = append(sdsps, dsp)
	}

	// Listen and connect
	if opts.ClientClient {
		logger.Printf("Connecting to %s\n", scs)
		sconn.Autoconnect(time.Duration(opts.Reconnect) * time.Second)
	} else {
		logger.Printf("Listening on %s\n", scs)
		sconn.Listen()
	}
	if opts.ServerServer {
		logger.Printf("Listening on %s\n", ccs)
		cconn.Listen()
	} else {
		logger.Printf("Connecting to %s\n", ccs)
		cconn.Autoconnect(time.Duration(opts.Reconnect) * time.Second)
	}

	// Set up signals to close nicely on Control+C
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, os.Kill)

	for interrupted := false; interrupted == false; {
		select {
		case p := <-sreceive:
			logger.Printf("S d:%02X p:%s\n", p.Dispatch(), p)
			cconn.Send(p)
		case p := <-creceive:
			logger.Printf("C d:%02X p:%s\n", p.Dispatch(), p)
			sconn.Send(p)
		case sig := <-signals:
			signal.Stop(signals)
			logger.Printf("signal %s\n", sig)
			sconn.Disconnect()
			cconn.Disconnect()
			interrupted = true
		}
	}

}
