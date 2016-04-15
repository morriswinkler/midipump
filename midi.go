package main

import (
	"encoding/csv"
	"os"
	"strconv"
	"strings"
	"time"

	syscall "golang.org/x/sys/unix"
)

type note struct {
	Id       int
	Note     byte
	Channel  byte
	Duration int
	State    bool
}

type midiNotes [36]note

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
			Id:       pump,
			Note:     byte(35 + pump),
			Channel:  0x0,
			Duration: duration,
			State:    true,
		}

		Info.Println(m[i])
	}

	return nil
}

func midiOut(receiver chan note) {

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

	var recv note
	for {
		recv = <-receiver
		Info.Printf("receive %#v\n", recv)

		if recv.State {
			command[0] = NoteOn + recv.Channel
			command[1] = recv.Note
			command[2] = 0x7f
		} else {
			command[0] = NoteOff + recv.Channel
			command[1] = recv.Note
			command[2] = 0x7f
		}

		if recv.State {
			Info.Printf("note %02d on \tduration %d\n", recv.Note, recv.Duration)
		} else {
			Info.Printf("note %02d off \tduration %d\n", recv.Note, recv.Duration)
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

func (p note) play(midiOutChan chan note) {
	midiOutChan <- p
	time.Sleep(time.Duration(p.Duration) * time.Millisecond)
	p.State = false
	midiOutChan <- p
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
	command0[8] = 0x24
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
