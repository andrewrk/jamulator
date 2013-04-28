package nes

import (
	"../asm6502"
	"io"
	"io/ioutil"
	"bufio"
	"fmt"
	"bytes"
	"os"
	"path"
	"strings"
	"strconv"
	"errors"
)

func (r *Rom) disassembleToDirWithJam(dest string, jamFd io.Writer) error {
	jam := bufio.NewWriter(jamFd)

	jam.WriteString("# output file name when this rom is assembled\n")
	jam.WriteString(fmt.Sprintf("filename=%s\n", r.Filename))
	jam.WriteString("# see http://wiki.nesdev.com/w/index.php/Mapper\n")
	jam.WriteString(fmt.Sprintf("mapper=%d\n", r.Mapper))
	jam.WriteString("# 'Horizontal', 'Vertical', or 'FourScreenVRAM'\n")
	jam.WriteString(fmt.Sprintf("mirroring=%s\n", r.Mirroring.String()))
	jam.WriteString("# whether SRAM in CPU $6000-$7FFF is present\n")
	jam.WriteString(fmt.Sprintf("sram=%t\n", r.SRamPresent))
	jam.WriteString("# whether the SRAM in CPU $6000-$7FFF, if present, is battery backed\n")
	jam.WriteString(fmt.Sprintf("battery=%t\n", r.BatteryBacked))
	jam.WriteString("# 'NTSC', 'PAL', or 'DualCompatible'\n")
	jam.WriteString(fmt.Sprintf("tvsystem=%s\n", r.TvSystem.String()))

	// save the prg rom
	jam.WriteString("# assembly code\n")
	for i, bank := range(r.PrgRom) {
		buf := bytes.NewBuffer(bank)
		program, err := asm6502.Disassemble(buf)
		if err != nil { return err }
		outpath := fmt.Sprintf("prg%d.asm", i)
		err = program.WriteSourceFile(path.Join(dest, outpath))
		if err != nil { return err }
		_, err = jam.WriteString(fmt.Sprintf("prg=%s\n", outpath))
		if err != nil { return err }
	}
	// save the chr banks
	jam.WriteString("# video data\n")
	for i, bank := range(r.ChrRom) {
		buf := bytes.NewBuffer(bank)
		outpath := fmt.Sprintf("chr%d.chr", i)
		chrFd, err := os.Create(path.Join(dest, outpath))
		if err != nil { return err }
		chr := bufio.NewWriter(chrFd)
		_, err = io.Copy(chr, buf)
		if err != nil {
			chrFd.Close()
			return err
		}
		_, err = jam.WriteString(fmt.Sprintf("chr=%s\n", outpath))
		if err != nil {
			chrFd.Close()
			return err
		}
		chr.Flush()
		err = chrFd.Close()
		if err != nil { return err }
	}

	jam.Flush()
	return nil
}

func (r *Rom) DisassembleToDir(dest string) error {
	// create the folder
	err := os.Mkdir(dest, 0770)
	if err != nil { return err }
	// put a .jam file which describes how to reassemble
	baseJamFilename := removeExtension(r.Filename)
	if len(baseJamFilename) == 0 {
		baseJamFilename = "rom"
	}
	jamFilename := path.Join(dest, baseJamFilename + ".jam")
	jamFd, err := os.Create(jamFilename)
	if err != nil { return err }

	err = r.disassembleToDirWithJam(dest, jamFd)
	err2 := jamFd.Close()
	if err != nil { return err }
	if err2 != nil { return err2 }
	return nil
}

func removeExtension(filename string) string {
	return filename[0:len(filename)-len(path.Ext(filename))]
}

func Assemble(dir string, ioreader io.Reader) (*Rom, error) {
	reader := bufio.NewReader(ioreader)
	r := new(Rom)
	r.PrgRom = make([][]byte, 0)
	r.ChrRom = make([][]byte, 0)

	lineCount := 0
	for {
		lineCount += 1
		rawLine, err := reader.ReadString('\n')
		if err == io.EOF { break }
		if err != nil { return nil, err }
		line := strings.TrimSpace(rawLine)
		if line[0] == '#' { continue }
		parts := strings.SplitN(line, "=", 2)
		if parts == nil {
			return nil, errors.New(fmt.Sprintf("Line %d: syntax error", lineCount))
		}
		switch parts[0] {
		case "filename":
			r.Filename = parts[1]
		case "mapper":
			m64, err := strconv.ParseUint(parts[1], 10, 8)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("Line %d: invalid mapper number: %d", lineCount, parts[1]))
			}
			r.Mapper = byte(m64)
		case "mirroring":
			switch parts[1] {
			case "Horizontal":
				r.Mirroring = HorizontalMirroring
			case "Vertical":
				r.Mirroring = VerticalMirroring
			case "FourScreenVRAM":
				r.Mirroring = FourScreenVRamMirroring
			default:
				return nil, errors.New(fmt.Sprintf("Line %d: unrecognized mirroring value: %s", lineCount, parts[1]))
			}
		case "tvsystem":
			switch parts[1] {
			case "NTSC":
				r.TvSystem = NtscTv
			case "PAL":
				r.TvSystem = PalTv
			case "DualCompatible":
				r.TvSystem = DualCompatTv
			default:
				return nil, errors.New(fmt.Sprintf("Line %d: unrecognized tvsystem value: %s", lineCount, parts[1]))
			}
		case "sram":
			switch parts[1] {
			case "true":
				r.SRamPresent = true
			case "false":
				r.SRamPresent = false
			default:
				return nil, errors.New(fmt.Sprintf("Line %d: unrecognized sram value: %s", lineCount, parts[1]))
			}
		case "battery":
			switch parts[1] {
			case "true":
				r.BatteryBacked = true
			case "false":
				r.BatteryBacked = false
			default:
				return nil, errors.New(fmt.Sprintf("Line %d: unrecognized battery value: %s", lineCount, parts[1]))
			}
		case "prg":
			prgfile := path.Join(dir, parts[1])
			programAst, err := asm6502.ParseFile(prgfile)
			if err != nil { return nil, err }
			program := programAst.ToProgram()
			if len(program.Errors) > 0 { return nil, program }
			bank := make([]byte, 0x4000)
			buf := bytes.NewBuffer(bank)
			err = program.Assemble(buf)
			if err != nil { return nil, err }
			r.PrgRom = append(r.PrgRom, bank)
		case "chr":
			chrfile := path.Join(dir, parts[1])
			chrFd, err := os.Open(chrfile)
			if err != nil { return nil, err }
			bank, err := ioutil.ReadAll(chrFd)
			err2 := chrFd.Close()
			if err != nil { return nil, err }
			if err2 != nil { return nil, err2 }
			r.ChrRom = append(r.ChrRom, bank)
		default:
			return nil, errors.New(fmt.Sprintf("Line %d: unrecognized property: %s", lineCount, parts[0]))
		}
	}

	return r, nil
}

func AssembleFile(filename string) (*Rom, error) {
	fd, err := os.Open(filename)
	if err != nil { return nil, err }

	r, err := Assemble(path.Dir(filename), fd)
	err2 := fd.Close()
	if err != nil { return nil, err }
	if err2 != nil { return nil, err2 }

	return r, nil
}
