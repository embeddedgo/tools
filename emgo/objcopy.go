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

func padBytes(cache *[]byte, n int, b byte) []byte {
	if len(*cache) < n {
		*cache = make([]byte, n)
		for i := range *cache {
			(*cache)[i] = b
		}
	}
	return (*cache)[:n]
}

func objcopy(elfFile, obj, format, incbin string) {
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
	if incbin != "" {
		sections = includeBins(sections, incbin)
	}
	switch format {
	case "bin", "z64":
		var w io.Writer
		if format == "z64" {
			w = bytes.NewBuffer(make([]byte, n64ChecksumLen))
		} else {
			f, err := os.Create(obj + "." + format)
			dieErr(err)
			defer f.Close()
			w = f
		}
		var ones []byte
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
			_, err = w.Write(padBytes(&ones, pad, 1))
			dieErr(err)
		}
		if format == "z64" {
			buf := w.(*bytes.Buffer)
			pad := n64ChecksumLen - buf.Len()
			if pad > 0 {
				buf.Write(padBytes(&ones, pad, 1))
			}
			//crc := n64CRC(buf.Bytes())
		}
	case "hex":
		w, err := os.Create(obj + ".hex")
		dieErr(err)
		defer w.Close()
		mem := gohex.NewMemory()
		for _, s := range sections {
			mem.AddBinary(uint32(s.addr), s.data)
		}
		mem.DumpIntelHex(w, 16)
	default:
		die("objcopy: unknown format: %s\n", format)
	}
}

/*
	Old code, without support for GOINCBIN, based on the external objcopy.

	objcopy, err := exec.LookPath("objcopy")
	dieErr(err)
	cmd = &exec.Cmd{
		Path:   objcopy,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	switch cfg["GOOUT"] {
	case "hex":
		cmd.Args = []string{objcopy, "-O", "ihex", elf, obj + ".hex"}
	case "bin":
		cmd.Args = []string{objcopy, "-O", "binary", elf, obj + ".bin"}
	default:
		die("unknown GOOUT: \"%s\"\n", cfg["GOOUT"])
	}
	if cmd.Run() != nil {
		os.Exit(1)
	}
*/
