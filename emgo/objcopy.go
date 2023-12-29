package main

import (
	"debug/elf"
	"fmt"
	"os"
	"sort"

	"github.com/marcinbor85/gohex"
)

func objcopy(elfFile, obj, goout string) {
	r, err := os.Open(elfFile)
	dieErr(err)
	defer r.Close()
	f, err := elf.NewFile(r)
	dieErr(err)
	defer f.Close()
	sections := make([]*elf.Section, 0, 10)
	for i, s := range f.Sections {
		if s.Type != elf.SHT_PROGBITS || s.Flags&elf.SHF_ALLOC == 0 {
			if k := i + 1; k < len(f.Sections) && len(sections) != 0 {
				n := f.Sections[k]
				if n.Type == elf.SHT_PROGBITS && n.Flags&elf.SHF_ALLOC != 0 {
					fmt.Fprintf(os.Stderr, "objcopy: skipping section '%s'\n", s.Name)
				}
			}
			continue
		}
		sections = append(sections, s)
	}
	if len(sections) == 0 {
		return
	}
	// Just in case sorting.
	sort.Slice(
		sections,
		func(i, j int) bool {
			return sections[i].Offset < sections[j].Offset
		},
	)
	switch goout {
	case "bin":
		w, err := os.Create(obj + ".bin")
		dieErr(err)
		defer w.Close()
		addr, offset := sections[0].Addr, sections[0].Offset
		var ones []byte
		for _, s := range sections {
			if n := s.Offset - offset; n != 0 {
				offset = s.Offset
				addr += n
				if len(ones) < int(n) {
					m := (n + 63) &^ 63
					ones = make([]byte, m, m)
					for i := range ones {
						ones[i] = 0xff
					}
				}
				_, err = w.Write(ones[:n])
				dieErr(err)
			}
			data, err := s.Data()
			dieErr(err)
			_, err = w.Write(data)
			dieErr(err)
			offset += s.Size
			addr += s.Size
		}
	case "hex":
		w, err := os.Create(obj + ".hex")
		dieErr(err)
		defer w.Close()
		startAddr, startOffset := sections[0].Addr, sections[0].Offset
		mem := gohex.NewMemory()
		for _, s := range sections {
			addr := uint32(startAddr + s.Offset - startOffset)
			data, err := s.Data()
			dieErr(err)
			mem.AddBinary(addr, data)
		}
		mem.DumpIntelHex(w, 32)
	default:
		die("objcopy: unknown GOOUT: \"%s\"\n", goout)
	}
}
