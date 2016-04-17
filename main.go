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
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
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

	midiDevice  = "/dev/ttyAMA0"
	rumbaDevice = "/dev/ttyACM0"
	logFile     = "/tmp/midipump.log"
)

var (
	midiNoteChan chan note // channel for note changes
	sseChan      chan note // channel to send server side events

	rumbaChan chan string // channel to connect to rumba

	// all pu,ps
	pumps midiNotes

	// logger
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger

	// command line args
	emulate    bool
	singlePump int

	// programm base path
	basePath string
)

func init() {

	flag.BoolVar(&emulate, "emulate", false, "enable hardware emulation")
	flag.IntVar(&singlePump, "pump", -1, "start single pump")

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

	// get the basePath
	var err error
	basePath, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		Error.Fatalf("could not read basedir, %s", err)
	}

}

func pumpSeq(notesChan chan note) {

	// TODO: figur out why 57 ... 64 do not work on the mideco board
	for i := range pumps {

		go pumps[i].play(midiNoteChan)
	}

}

func pumpAllStart(notesChan chan note) {

	for i := range pumps {
		pumps[i].State = true
		notesChan <- pumps[i]
	}

}

func pumpAllStop(notesChan chan note) {

	for i := range pumps {
		pumps[i].State = false
		notesChan <- pumps[i]
	}

	// reread CSV file
	err := loadCsv()
	if err != nil {
		Error.Printf("unable to load csv file: %s", err)
	}
}

func pumpSingle(i int, notesChan chan note) {

	notesChan <- pumps[i]
}

func loadCsv() error {

	filepath := getBasePath("csv/upload.csv")
	err := pumps.readCsvFile(filepath)
	if err != nil {
		return err
	}
	return nil
}

func getBasePath(file string) string {
	s := []string{basePath, file}
	res := strings.Join(s, "/")
	return res
}

func main() {

	// read command line arguments
	flag.Parse()

	// load CSV File
	err := loadCsv()
	if err != nil {
		Error.Printf("unable to load csv file: %s", err)
	}

	midiNoteChan = make(chan note)
	sseChan = make(chan note)

	rumbaChan = make(chan string)

	go midiOut(midiNoteChan)
	go rumba(rumbaChan)

	if singlePump != -1 {
		pumpSingle(singlePump, midiNoteChan)
	}

	serverEvents(sseChan)

}
