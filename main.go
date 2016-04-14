/*

MIDI messages are comprised of two components: commands and data bytes.
The command byte tells the MIDI instrument what type of message is being
sent and the subsequent data byte(s) store the actual data. For example
a command byte might tell a MIDI instrument that it going to send information
about pitchbend, and the data byte describes how much pitchbend.

MIDI data bytes range from 0 to 127. Convert these numbers to binary and we
see they range from 00000000 to 01111111, the important thing to notice here
is that they always start with a 0 as the most significant bit (MSB). MIDI
command bytes range from 128 to 255, or 1000000 to 11111111 in binary. Unlike
data bytes, MIDI command bytes always start with a 1 as the MSB. This MSB is
how a MIDI instrument differentiates between a command byte and a data byte.

MIDI commands are further broken down by the following system:

The first half of the MIDI command byte (the three bits following the MSB) sets
the type of command. More info about the meaning on each of these commands is here.

10000000 = note off
10010000 = note on
10100000 = aftertouch
10110000 = continuous controller
11000000 = patch change
11010000 = channel pressure
11100000 = pitch bend
11110000 = non-musical commands

The last half of the command byte sets the MIDI channel. All the bytes
listed above would be in channel 0, command bytes ending in 0001 would
be for MIDI channel 1, and so on.

All MIDI messages start with a command byte, some messages contain one
data byte, others contain two or more (see image above). For example, a
note on command byte is followed by two data bytes: note and velocity.
I
'm going to explain how to use note on, note off, velocity, and pitchbend
in this instructable, since these are the most commonly used commands.
I'm sure you will be able to infer how to set up the others by the end of this.

*/

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"unsafe"

	"flag"
	"strconv"
	"strings"

	syscall "golang.org/x/sys/unix"

	"net/http"

	"github.com/janberktold/sse"
)

const (
	UartBase = 0x20201000

	UARTFR = 0x18
	IBRD   = 0x24
	FBRD   = 0x28
	LCRH   = 0x2C
	UARTCR = 0x30
)

type Mem struct {
	Map []byte
	sync.Mutex
}

func (m *Mem) Open() (err error) {

	file, err := os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0)
	if err != nil {
		err = errors.New(fmt.Sprintf("Error open /dev/mem", err))
		return
	}
	// for mmap file can be closed after memory mapping is setup
	defer file.Close()

	// samaphore
	m.Lock()
	defer m.Unlock()

	//r.Mem.Map, err = mmap.Map(r.Mem.Fd, r.Mem.Offset, r.Mem.Size, mmap.PROT_READ|mmap.PROT_WRITE, mmap.MAP_SHARED)
	// mmap call to map the uart clock registers
	m.Map, err = syscall.Mmap(
		int(file.Fd()),
		UartBase,
		4096,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED)

	if err != nil {
		err = errors.New(fmt.Sprintf("Error mmap ", err))
		return err
	}

	return nil
}

func setBaudrate() {

	var m Mem

	err := m.Open()
	if err != nil {
		fmt.Println(err)
	}

	m.Lock()
	defer m.Unlock()

	var rate uint32

	ebuf := bytes.NewReader(m.Map[IBRD:(IBRD + 0x04)])
	err = binary.Read(ebuf, binary.LittleEndian, &rate)
	if err != nil {
		Error.Printf("binary.Read failed:\n", err)
	}

	Info.Println(rate)

	*(*uint32)(unsafe.Pointer(&m.Map[UARTCR])) = 0x00

	*(*uint32)(unsafe.Pointer(&m.Map[IBRD])) = uint32(6)
	*(*uint32)(unsafe.Pointer(&m.Map[FBRD])) = 0
	*(*uint32)(unsafe.Pointer(&m.Map[LCRH])) = 0x70
	*(*uint32)(unsafe.Pointer(&m.Map[UARTCR])) = 0x0301

	ebuf = bytes.NewReader(m.Map[IBRD:(IBRD + 0x04)])
	err = binary.Read(ebuf, binary.LittleEndian, &rate)
	if err != nil {
		Error.Printf("binary.Read failed:\n", err)
	}

	Info.Println(rate)
}

func midiReset(serial *os.File) {

	command0 := make([]byte, 10)
	command0[0] = 0xf0
	command0[1] = 0x00
	command0[2] = 0x20
	command0[3] = 0x7a
	command0[4] = 0x05
	command0[5] = 0x01
	command0[6] = 0x01
	command0[7] = 0x01
	command0[8] = 0x2a
	command0[9] = 0xf7

	command1 := make([]byte, 10)
	command1[0] = 0xf0
	command1[1] = 0x00
	command1[2] = 0x20
	command1[3] = 0x7a
	command1[4] = 0x05
	command1[5] = 0x04
	command1[6] = 0x00
	command1[7] = 0xf7

	command2 := make([]byte, 10)
	command2[0] = 0xf0
	command2[1] = 0x00
	command2[2] = 0x20
	command2[3] = 0x7a
	command2[4] = 0x05
	command2[5] = 0x02
	command2[6] = 0x05
	command2[7] = 0xf7

	_, err := serial.Write(command0)
	if err != nil {
		Error.Printf("coul not write to serial port %s err: %s", midiDevice, err)
	}

	_, err = serial.Write(command1)
	if err != nil {
		Error.Printf("coul not write to serial port %s err: %s", midiDevice, err)
	}

	_, err = serial.Write(command2)
	if err != nil {
		Error.Printf("coul not write to serial port %s err: %s", midiDevice, err)
	}

}

const (
	NoteOff         = 0x80
	NoteOn          = 0x90
	Aftertouch      = 0xa0
	ContinuousContr = 0xb0
	PatchChange     = 0xc0
	ChannelPressure = 0xD0
	PitchBend       = 0xE0
	SysExC          = 0xF0

	midiDevice = "/dev/ttyAMA0"
	logFile    = "/tmp/midipump.log"
)

var (
	midiNoteChan chan *note // channel for note changes
	sseChan      chan *note // channel to send server side events

	pumps midiNotes
	wg    sync.WaitGroup

	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger

	emulate bool = false
)

type note struct {
	note     byte
	channel  byte
	duration int
	state    bool
}

type midiNotes [32]note

func (m *midiNotes) readCsvFile(file string) (err error) {

	csvfile, err := os.Open(file)

	if err != nil {
		return err
	}
	defer csvfile.Close()

	reader := csv.NewReader(csvfile)

	reader.FieldsPerRecord = 2 // see the Reader struct information below

	rawCSVdata, err := reader.ReadAll()

	if err != nil {
		return err
	}

	// sanity check, display to standard output
	for i, v := range rawCSVdata {
		pump, err := strconv.Atoi(strings.TrimSpace(v[0]))
		duration, err := strconv.Atoi(strings.TrimSpace(v[1]))
		if err != nil {
			return err
		}

		Info.Printf("pump : %d with duration : %d ms\n", pump, duration)

		m[i] = note{
			note:     byte(36 + pump),
			channel:  0x0,
			duration: duration,
			state:    true,
		}
	}

	return nil
}

func midiOut(receiver chan *note) {

	var serial *os.File
	var err error
	// just initialize the serial port and the mideco if emulate=false
	if !emulate {

		serial, err = os.OpenFile(midiDevice, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
		if err != nil {
			Error.Printf("coul not open serial port %s err: %s", midiDevice, err)
		}
		defer serial.Close()

		setBaudrate()
		midiReset(serial)
	}

	command := make([]byte, 3)

	for {
		recv := <-receiver

		if recv.state {
			command[0] = NoteOn + recv.channel
			command[1] = recv.note
			command[2] = 0x7f
		} else {
			command[0] = NoteOff + recv.channel
			command[1] = recv.note
			command[2] = 0x7f
		}

		if recv.state {
			Info.Printf("note %02d on \tduration %d\n", recv.note, recv.duration)
		} else {
			Info.Printf("note %02d off \tduration %d\n", recv.note, recv.duration)
		}

		Info.Printf("command %#v \n", command)

		if !emulate {
			_, err = serial.Write(command)
			if err != nil {
				Error.Printf("coul not write to serial port %s err: %s", midiDevice, err)
			}
		}

	}
}

func (p *note) play(midiOutChan chan *note) {

	midiOutChan <- p
	time.Sleep(time.Duration(p.duration) * time.Millisecond)
	p.state = false
	midiOutChan <- p
	wg.Done()

}

func init() {

	flag.BoolVar(&emulate, "emulate", false, "enable hardware emulation")

	// fdHandl, err := os.OpenFile("file.txt”, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	// if err != nil {
	// 	log.Fatalln("Failed to open log file”, output, ":", err)
	// }

	fdHandle := os.Stdout

	Trace = log.New(fdHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(fdHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(fdHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(fdHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)

}

func pumpAll(notesChan chan *note) {

	// TODO: figur out why 57 ... 64 do not work on the mideco board
	Info.Println("i am here")
	for i := range pumps {

		wg.Add(1)
		go pumps[i].play(midiNoteChan)
	}
	wg.Wait()

}

func main() {

	// read command line arguments
	flag.Parse()

	err := pumps.readCsvFile("csv/example.csv")
	if err != nil {
		Error.Printf("error reading csv file: %s", err)
	}

	midiNoteChan = make(chan *note)
	sseChan = make(chan *note)

	go midiOut(midiNoteChan)

	serverEvents(sseChan)

}

func serverEvents(msgChan chan *note) {

	http.HandleFunc("/event", func(w http.ResponseWriter, r *http.Request) {
		// get a SSE connection from the HTTP request
		// in a real world situation, you should look for the error (second return value)
		conn, _ := sse.Upgrade(w, r)

		for {
			// keep this goroutine alive to keep the connection

			// get a message from some channel
			// blocks until it recieves a messages and then instantly sends it to the client
			msg := <-msgChan
			Info.Println(msg)
			conn.WriteJson(struct {
				Id   int
				Note *note
			}{
				Id:   int(msg.note),
				Note: msg,
			})
		}
	})

	http.HandleFunc("/pump", func(w http.ResponseWriter, r *http.Request) {
		go pumpAll(midiNoteChan)

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, ("./static" + r.URL.Path))
	})

	http.ListenAndServe(":8080", nil)

}
