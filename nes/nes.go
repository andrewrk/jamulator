package nes

import (
	"os"
	"bufio"
	"io"
	"errors"
)

const (
	HorizontalMirroring = iota
	VerticalMirroring
	FourScreenVRamMirroring
)

const (
	NtscTv = iota
	PalTv
	DualCompatTv
)

type Rom struct {
	PrgBankCount int // Size of PRG ROM in 16 KB units
	ChrBankCount int // Size of CHR ROM in 8 KB units (Value 0 means the board uses CHR RAM)
	PrgRamSize int // Size of PRG RAM in 8 KB units (Value 0 infers 8 KB for compatibility)

	Mapper byte
	Mirroring int
	BatteryBacked bool
	TvSystem int
	SRamPresent bool
}

func Disassemble(ioreader io.Reader) (*Rom, error) {
	reader := bufio.NewReader(ioreader)
	r := new(Rom)

	// read the header
	buf := make([]byte, 16)
	_, err := reader.Read(buf)
	if err != nil { return nil, err }
	if string(buf[0:4]) != "NES\x1a" {
		return nil, errors.New("Invalid ROM file")
	}
	r.PrgBankCount = int(buf[4])
	r.ChrBankCount = int(buf[5])
	flags6 := buf[6]
	flags7 := buf[7]
	r.PrgRamSize = int(buf[8])
	flags9 := buf[9]
	flags10 := buf[10]

	r.Mapper = (flags6 >> 4) | (flags7 & 0xf0)
	if flags6 & 0x8 != 0 {
		r.Mirroring = FourScreenVRamMirroring
	} else if flags6 & 0x1 != 0 {
		r.Mirroring = VerticalMirroring
	} else {
		r.Mirroring = HorizontalMirroring
	}
	if flags6 & 0x2 != 0 {
		r.BatteryBacked = true
	}
	if flags6 & 0x4 != 0 {
		return nil, errors.New("Trainer unsupported")
	}
	if flags7 & 0x1 != 0 {
		return nil, errors.New("VS Unisystem unsupported")
	}
	if flags7 & 0x2 != 0 {
		return nil, errors.New("PlayChoice-10 unsupported")
	}
	if (flags7 >> 2) & 0x2 != 0 {
		return nil, errors.New("NES 2.0 format unsupported")
	}
	if flags9 & 0x1 != 0 {
		return nil, errors.New("PAL unsupported")
	}
	switch flags10 & 0x2 {
	case 0: r.TvSystem = NtscTv
	case 2: r.TvSystem = PalTv
	default: r.TvSystem = DualCompatTv
	}
	r.SRamPresent = flags10 & 0x10 == 0
	if flags10 & 0x20 != 0 {
		return nil, errors.New("bus conflicts unsupported")
	}
	return r, nil
}

func DisassembleFile(filename string) (*Rom, error) {
	fd, err := os.Open(filename)
	if err != nil { return nil, err }

	r, err := Disassemble(fd)
	err2 := fd.Close()
	if err != nil { return nil, err }
	if err2 != nil { return nil, err2 }

	return r, nil
}
