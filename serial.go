package main

import (
	"errors"
	"fmt"
	"os"
	"sync"

	syscall "golang.org/x/sys/unix"

	"bytes"
	"encoding/binary"
	"unsafe"
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
