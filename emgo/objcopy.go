package main

import (
	"bytes"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/marcinbor85/gohex"
)

type section struct {
	addr   uint64
	offset int64
	data   []byte
}

func includeBins(sections []*section, incbin string) []*section {
	for _, ba := range strings.Split(incbin, ",") {
		i := strings.IndexByte(ba, ':')
		if i <= 0 {
			die("objcopy: bad '%s' in GOINCBIN (format BIN1:ADDR1,BIN2:ADDR2,...)", ba)
		}
		bin, addr := ba[:i], ba[i+1:]
		s := new(section)
		var err error
		s.addr, err = strconv.ParseUint(addr, 0, 64)
		if err != nil {
			die("objcopy: bad address in '%s': %s\n", addr, err)
		}
		s.data, err = os.ReadFile(bin)
		if err != nil && errors.Is(err, fs.ErrNotExist) && filepath.Dir(bin) == "." {
			s.data, err = os.ReadFile(filepath.Join("..", bin))
		}
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				die("objdump: cannot find %s in . or ..\n", bin)
			} else {
				dieErr(err)
			}
		}
		first, last := sections[0], sections[len(sections)-1]
		if o := int64(first.addr - s.addr); o >= int64(len(s.data)) {
			s.offset = first.offset - o
			sections = append([]*section{s}, sections...)
		} else if o = int64(s.addr - last.addr); o >= int64(len(last.data)) {
			s.offset = last.offset + o
			sections = append(sections, s)
		} else {
			die("objcopy: %s overlaps with the current image", bin)
		}
	}
	return sections
}

var ones []byte

func padBytes(cache *[]byte, n int, b byte) []byte {
	if len(*cache) < n {
		*cache = make([]byte, n)
		for i := range *cache {
			(*cache)[i] = b
		}
	}
	return (*cache)[:n]
}

func objcopy(elfFile, obj string, cfg map[string]string) {
	r, err := os.Open(elfFile)
	dieErr(err)
	defer r.Close()
	f, err := elf.NewFile(r)
	dieErr(err)
	defer f.Close()
	sections := make([]*section, 0, 10)
	for i, s := range f.Sections {
		if s.Type != elf.SHT_PROGBITS || s.Flags&elf.SHF_ALLOC == 0 {
			if k := i + 1; k < len(f.Sections) && len(sections) != 0 {
				n := f.Sections[k]
				if n.Type == elf.SHT_PROGBITS && n.Flags&elf.SHF_ALLOC != 0 {
					fmt.Fprintf(os.Stderr, "objcopy: skipping section '%s' (%d bytes)\n", s.Name, s.Size)
				}
			}
			continue
		}
		data, err := s.Data()
		dieErr(err)
		sections = append(sections, &section{s.Addr, int64(s.Offset), data})
	}
	if len(sections) == 0 {
		return
	}
	sort.Slice(
		sections,
		func(i, j int) bool {
			return sections[i].offset < sections[j].offset
		},
	)
	startAddr, startOffset := sections[0].addr, sections[0].offset
	for _, s := range sections {
		s.offset -= startOffset
		s.addr = startAddr + uint64(s.offset)
	}
	if incbin := cfg["GOINCBIN"]; incbin != "" {
		sections = includeBins(sections, incbin)
	}
	switch format := cfg["GOOUT"]; format {
	case "bin", "z64", "uf2":
		var w io.Writer
		switch cfg["GOTARGET"] {
		case "n64", "rp2350":
			w = bytes.NewBuffer(make([]byte, 0, n64ChecksumLen))
		default:
			if format != "bin" {
				die(
					"objcopy: %s format not supported for GOTARGET=%s",
					format, cfg["GOTARGET"],
				)
			}
			f, err := os.Create(obj + ".bin")
			dieErr(err)
			defer f.Close()
			w = f
		}
		for i, s := range sections {
			_, err = w.Write(s.data)
			dieErr(err)
			pad := 0
			if i+1 < len(sections) {
				pad = int(sections[i+1].offset-s.offset) - len(s.data)
			}
			if pad == 0 {
				continue
			}
			_, err = w.Write(padBytes(&ones, pad, 0xff))
			dieErr(err)
		}
		switch cfg["GOTARGET"] {
		case "n64":
			n64WriteROMFile(obj, format, w.(*bytes.Buffer))
		case "rp2350":
			picoImage(obj, format, w.(*bytes.Buffer))
		}
	case "hex":
		w, err := os.Create(obj + ".hex")
		dieErr(err)
		defer w.Close()
		mem := gohex.NewMemory()
		for _, s := range sections {
			mem.AddBinary(uint32(s.addr), s.data)
		}
		dieErr(mem.DumpIntelHex(w, 16))
	default:
		die("objcopy: unknown format: %s\n", format)
	}
}

func dieFormat(format, target string) {
	die("objcopy: %s format not supported for GOTARGET=%s", format, target)
}
