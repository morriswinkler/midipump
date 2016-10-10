package main

import (
	"errors"
	"fmt"
	"os"
	"sync"

	syscall "golang.org/x/sys/unix"

	"bytes"
	"encoding/binary"
	"time"
	"unsafe"
)

const (
	UartBase = 0x20201000 // UART register base address

	UARTFR = 0x18 // flag reg (RO)
	IBRD   = 0x24 // integer baud rate register
	FBRD   = 0x28 // fractional baud rate register
	LCRH   = 0x2C // Line control register (also called UARTLCR_H in the PL011 docs)
	UARTCR = 0x30 // UART control register
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
	// mmap file can be closed after memory mapping is setup
	defer file.Close()

	// semaphore
	m.Lock()
	defer m.Unlock()

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

	// current rate
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

	// new rate 31250
	Info.Println(rate)
}

func openSerialRumba(deviceFile string) (f *os.File, err error) {

	f, err = os.OpenFile(deviceFile, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NONBLOCK, 0666)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil && f != nil {
			f.Close()
		}
	}()

	fd := f.Fd()
	vmin, vtime := posixTimeoutValues(time.Duration(time.Second * 5))
	t := syscall.Termios{
		Iflag:  syscall.IGNPAR,
		Cflag:  syscall.CS8 | syscall.CREAD | syscall.CLOCAL | uint32(0x1002),
		Cc:     [19]uint8{syscall.VMIN: vmin, syscall.VTIME: vtime},
		Ispeed: uint32(0x1002),
		Ospeed: uint32(0x1002),
	}

	if _, _, errno := syscall.Syscall6(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(syscall.TCSETS),
		uintptr(unsafe.Pointer(&t)),
		0,
		0,
		0,
	); errno != 0 {
		return nil, errno
	}

	if err = syscall.SetNonblock(int(fd), false); err != nil {
		return
	}

	return f, nil
}

func posixTimeoutValues(readTimeout time.Duration) (vmin uint8, vtime uint8) {
	const MAXUINT8 = 1<<8 - 1 // 255
	// set blocking / non-blocking read
	var minBytesToRead uint8 = 1
	var readTimeoutInDeci int64
	if readTimeout > 0 {
		// EOF on zero read
		minBytesToRead = 0
		// convert timeout to deciseconds as expected by VTIME
		readTimeoutInDeci = (readTimeout.Nanoseconds() / 1e6 / 100)
		// capping the timeout
		if readTimeoutInDeci < 1 {
			// min possible timeout 1 Deciseconds (0.1s)
			readTimeoutInDeci = 1
		} else if readTimeoutInDeci > MAXUINT8 {
			// max possible timeout is 255 deciseconds (25.5s)
			readTimeoutInDeci = MAXUINT8
		}
	}
	return minBytesToRead, uint8(readTimeoutInDeci)
}
