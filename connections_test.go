// Author  Raido Pahtma
// License MIT

package moteconnection

import "testing"
import "fmt"

func TestConnectionParser(t *testing.T) {
	conn, _, err := CreateConnection("sf@host:9000")
	if err != nil {
		t.Error(err)
	} else if conn.(*SfConnection).Host != "host" || conn.(*SfConnection).Port != 9000 {
		t.Error(fmt.Sprintf("error %s %d", conn.(*SfConnection).Host, conn.(*SfConnection).Port))
	}

	conn, _, err = CreateConnection("sf@host")
	if err != nil {
		t.Error(err)
	} else if conn.(*SfConnection).Host != "host" || conn.(*SfConnection).Port != 9002 {
		t.Error(fmt.Sprintf("error %s %d", conn.(*SfConnection).Host, conn.(*SfConnection).Port))
	}

	conn, _, err = CreateConnection("")
	if err != nil {
		t.Error(err)
	} else if conn.(*SfConnection).Host != "localhost" || conn.(*SfConnection).Port != 9002 {
		t.Error(fmt.Sprintf("error %s %d", conn.(*SfConnection).Host, conn.(*SfConnection).Port))
	}

	conn, _, err = CreateConnection("sf@host:")
	if err == nil {
		t.Error(err)
	}

	conn, _, err = CreateConnection("sf@host:a")
	if err == nil {
		t.Error(err)
	}

	conn, _, err = CreateConnection("sf@/*?")
	if err == nil {
		t.Error(err)
	}

	conn, _, err = CreateConnection("COM0:115200")
	if err == nil {
		t.Error(err)
	}

	conn, _, err = CreateConnection("sf@COM0:115200")
	if err == nil {
		t.Error(err)
	}

	conn, _, err = CreateConnection("serial@COM0:115200")
	if err != nil {
		t.Error(err)
	} else if conn.(*SerialConnection).Port != "COM0" || conn.(*SerialConnection).Baud != 115200 {
		t.Error(fmt.Sprintf("error %s %d", conn.(*SerialConnection).Port, conn.(*SerialConnection).Baud))
	}

	conn, _, err = CreateConnection("serial@COM0")
	if err != nil {
		t.Error(err)
	} else if conn.(*SerialConnection).Port != "COM0" || conn.(*SerialConnection).Baud != 115200 {
		t.Error(fmt.Sprintf("error %s %d", conn.(*SerialConnection).Port, conn.(*SerialConnection).Baud))
	}

	conn, _, err = CreateConnection("serial@COM0:9600")
	if err != nil {
		t.Error(err)
	} else if conn.(*SerialConnection).Port != "COM0" || conn.(*SerialConnection).Baud != 9600 {
		t.Error(fmt.Sprintf("error %s %d", conn.(*SerialConnection).Port, conn.(*SerialConnection).Baud))
	}

	conn, _, err = CreateConnection("serial@/dev/ttyUSB0:115200")
	if err != nil {
		t.Error(err)
	} else if conn.(*SerialConnection).Port != "/dev/ttyUSB0" || conn.(*SerialConnection).Baud != 115200 {
		t.Error(fmt.Sprintf("error %s %d", conn.(*SerialConnection).Port, conn.(*SerialConnection).Baud))
	}

	conn, _, err = CreateConnection("serial@/dev/ttyUSB0")
	if err != nil {
		t.Error(err)
	} else if conn.(*SerialConnection).Port != "/dev/ttyUSB0" || conn.(*SerialConnection).Baud != 115200 {
		t.Error(fmt.Sprintf("error %s %d", conn.(*SerialConnection).Port, conn.(*SerialConnection).Baud))
	}

	conn, _, err = CreateConnection("serial@/dev/ttyUSB0:9600")
	if err != nil {
		t.Error(err)
	} else if conn.(*SerialConnection).Port != "/dev/ttyUSB0" || conn.(*SerialConnection).Baud != 9600 {
		t.Error(fmt.Sprintf("error %s %d", conn.(*SerialConnection).Port, conn.(*SerialConnection).Baud))
	}

	conn, _, err = CreateConnection("serial@COM0:")
	if err == nil {
		t.Error(err)
	}

	conn, _, err = CreateConnection("serial@COM1:a")
	if err == nil {
		t.Error(err)
	}

	conn, _, err = CreateConnection("serial@/*?")
	if err == nil {
		t.Error(err)
	}

	conn, _, err = CreateConnection("sf@")
	if err == nil {
		t.Error(err)
	}

}
