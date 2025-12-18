// SF server listener - listens for incoming SF connections and
// prints all incoming packets. Not a practical tool, but perhaps useful
// for testing things out.

// @author Raido Pahtma
// @license MIT

package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/jessevdk/go-flags"
	"github.com/proactivity-lab/go-moteconnection"
)

// ApplicationVersionMajor -
const ApplicationVersionMajor = 0

// ApplicationVersionMinor -
const ApplicationVersionMinor = 0

// ApplicationVersionPatch -
const ApplicationVersionPatch = 0

// ApplicationBuildDate -
var ApplicationBuildDate string

// ApplicationBuildDistro -
var ApplicationBuildDistro string

func main() {

	var opts struct {
		Positional struct {
			ConnectionString string `description:"Connectionstring sf@0.0.0.0:PORT, default sf@0.0.0.0:9002"`
		} `positional-args:"yes"`

		ActiveMessage bool   `long:"am" description:"Configure an ActiveMessage dispatcher"`
		Debug         []bool `short:"D" long:"debug" description:"Debug mode, print raw packets"`
		ShowVersion   func() `short:"V" long:"version" description:"Show application version"`
	}

	opts.ShowVersion = func() {
		if ApplicationBuildDate == "" {
			ApplicationBuildDate = "YYYY-mm-dd_HH:MM:SS"
		}
		if ApplicationBuildDistro == "" {
			ApplicationBuildDistro = "unknown"
		}
		fmt.Printf("sfserver %d.%d.%d (%s %s)\n",
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

	if len(opts.Positional.ConnectionString) == 0 {
		opts.Positional.ConnectionString = "sf@0.0.0.0:9002"
	}

	conn, cs, err := moteconnection.CreateConnection(opts.Positional.ConnectionString)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		os.Exit(1)
	}

	receive := make(chan moteconnection.Packet)
	if opts.ActiveMessage {
		dsp := moteconnection.NewMessageDispatcher(moteconnection.NewMessage(0, 0))
		dsp.RegisterMessageSnooper(receive)
		conn.AddDispatcher(dsp)
	} else {
		for i := 0; i <= 255; i++ {
			dsp := moteconnection.NewPacketDispatcher(moteconnection.NewRawPacket(byte(i)))
			dsp.RegisterReceiver(receive)
			conn.AddDispatcher(dsp)
		}
	}

	// Configure logging
	logformat := log.Ldate | log.Ltime | log.Lmicroseconds
	var logger *log.Logger
	if len(opts.Debug) > 0 {
		if len(opts.Debug) > 1 {
			logformat = logformat | log.Lshortfile
		}
		logger = log.New(os.Stdout, "INFO:  ", logformat)
		conn.SetDebugLogger(log.New(os.Stdout, "DEBUG: ", logformat))
		conn.SetInfoLogger(logger)
	} else {
		logger = log.New(os.Stdout, "", logformat)
	}
	conn.SetWarningLogger(log.New(os.Stdout, "WARN:  ", logformat))
	conn.SetErrorLogger(log.New(os.Stdout, "ERROR: ", logformat))

	// Connect to the host
	logger.Printf("Listening on %s\n", cs)
	conn.Listen()

	// Set up signals to close nicely on Control+C
	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, os.Kill)

	for interrupted := false; interrupted == false; {
		select {
		case msg := <-receive:
			logger.Printf("%s\n", msg)
		case sig := <-signals:
			signal.Stop(signals)
			logger.Printf("signal %s\n", sig)
			conn.Disconnect()
			interrupted = true
		}
	}

}
