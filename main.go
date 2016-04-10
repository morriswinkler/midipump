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

I'm going to explain how to use note on, note off, velocity, and pitchbend
in this instructable, since these are the most commonly used commands.
I'm sure you will be able to infer how to set up the others by the end of this.

*/

package main

import (
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	syscall "golang.org/x/sys/unix"
)

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
	midiNoteChan chan *note
	midiNotes    [64]note
	wg           sync.WaitGroup

	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

type note struct {
	note     byte
	channel  byte
	duration int
	state    bool
}

func midiOut(receiver chan *note) {

	serial, err := os.OpenFile(midiDevice, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		Error.Printf("coul not open serial port %s err: %s", midiDevice, err)
	}
	defer serial.Close()

	command := make([]byte, 3)

	for {
		recv := <-receiver

		if recv.state {
			command[0] = NoteOn + recv.channel
			command[1] = recv.note
			command[2] = 0xff
		} else {
			command[0] = NoteOn + recv.channel
			command[1] = recv.note
			command[2] = 0xff
		}

		if recv.state {
			Info.Printf("note %02d on \tduration %d\n", recv.note, recv.duration)
		} else {
			Info.Printf("note %02d off \tduration %d\n", recv.note, recv.duration)
		}

		Info.Printf("command %016b \n", command)
		_, err = serial.Write(command)
		if err != nil {
			Error.Printf("coul not write to serial port %s err: %s", midiDevice, err)
		}

	}
}

func (p *note) play(midiOutChan chan *note) {

	midiOutChan <- p
	time.Sleep(time.Duration(p.duration) * time.Second)
	p.state = false
	midiOutChan <- p
	wg.Done()

}

func init() {

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

func main() {

	_ = rand.New(rand.NewSource(99))
	midiNoteChan = make(chan *note)

	go midiOut(midiNoteChan)

	midiNotes[0] = note{
		note:     byte(0),
		channel:  0x1,
		duration: 10,
		state:    true,
	}
	wg.Add(1)
	go midiNotes[0].play(midiNoteChan)

	wg.Wait()
}
