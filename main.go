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
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	NoteOff         = 0x80 << 8
	NoteOn          = 0x90 << 8
	Aftertouch      = 0xa0 << 8
	ContinuousContr = 0xb0 << 8
	PatchChange     = 0xc0 << 8
	ChannelPressure = 0xD0 << 8
	PitchBend       = 0xE0 << 8
	SysExC          = 0xF0 << 8
)

var (
	midiNoteChan chan *note
	midiNotes    [64]note
	wg           sync.WaitGroup
)

type note struct {
	note     int
	channel  int
	duration int
	state    bool
}

func midiOut(receiver chan *note) {

	var recv *note
	var command uint16
	for {
		recv = <-receiver

		if recv.state {
			command = NoteOn + uint16(recv.channel)<<8 + uint16(recv.note)
		} else {
			command = NoteOff + uint16(recv.channel)<<8 + uint16(recv.note)
		}

		if recv.state {
			fmt.Printf("note %02d on \tduration %d\n", recv.note, recv.duration)
		} else {
			fmt.Printf("note %02d off \tduration %d\n", recv.note, recv.duration)
		}

		fmt.Printf("command %016b \n", command)
	}
}

func (p *note) play(midiOutChan chan *note) {

	midiOutChan <- p
	time.Sleep(time.Duration(p.duration) * time.Second)
	p.state = false
	midiOutChan <- p
	wg.Done()

}

func main() {

	r := rand.New(rand.NewSource(99))
	midiNoteChan = make(chan *note)

	go midiOut(midiNoteChan)

	for i := range midiNotes {

		midiNotes[i] = note{
			note:     i,
			channel:  5,
			duration: r.Intn(20),
			state:    true,
		}
		wg.Add(1)
		go midiNotes[i].play(midiNoteChan)
	}

	wg.Wait()
}
