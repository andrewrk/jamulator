package nes

import (
	"../asm6502"
	"io"
	"bufio"
	"fmt"
	"bytes"
	"os"
	"path"
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
