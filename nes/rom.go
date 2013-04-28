package nes

import (
	"os"
	"bufio"
	"io"
	"errors"
	"path"
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

type TvSystem int

func (tvs TvSystem) String() string {
	if tvs == NtscTv {
		return "NTSC"
	} else if tvs == PalTv {
		return "PAL"
	}
	return "DualCompatible"
}

type Mirroring int

func (m Mirroring) String() string {
	if m == HorizontalMirroring {
		return "Horizontal"
	} else if m == VerticalMirroring {
		return "Vertical"
	}
	return "FourScreenVRAM"
}

type Rom struct {
	Filename string
	PrgRom [][]byte
	ChrRom [][]byte

	Mapper byte
	Mirroring Mirroring
	BatteryBacked bool
	TvSystem TvSystem
	SRamPresent bool
}


func Load(ioreader io.Reader) (*Rom, error) {
	reader := bufio.NewReader(ioreader)
	r := new(Rom)

	// read the header
	buf := make([]byte, 16)
	_, err := io.ReadAtLeast(reader, buf, 16)
	if err != nil { return nil, err }
	if string(buf[0:4]) != "NES\x1a" {
		return nil, errors.New("Invalid ROM file")
	}
	prgBankCount := int(buf[4])
	chrBankCount := int(buf[5])
	flags6 := buf[6]
	flags7 := buf[7]
	if buf[8] != 0 && buf[8] != 1 {
		return nil, errors.New("Only 8KB program RAM supported")
	}
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

	r.PrgRom = make([][]byte, prgBankCount)
	for i := 0; i < prgBankCount; i++ {
		bank := make([]byte, 0x4000)
		_, err := io.ReadAtLeast(reader, bank, len(bank))
		if err != nil { return nil, err }
		r.PrgRom[i] = bank
	}

	r.ChrRom = make([][]byte, chrBankCount)
	for i := 0; i < chrBankCount; i++ {
		bank := make([]byte, 0x2000)
		_, err := io.ReadAtLeast(reader, bank, len(bank))
		if err != nil { return nil, err }
		r.ChrRom[i] = bank
	}


	return r, nil
}

func LoadFile(filename string) (*Rom, error) {
	fd, err := os.Open(filename)
	if err != nil { return nil, err }

	r, err := Load(fd)
	r.Filename = path.Base(filename)
	err2 := fd.Close()
	if err != nil { return nil, err }
	if err2 != nil { return nil, err2 }

	return r, nil
}

