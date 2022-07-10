// Copyright 2022 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/build"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	cfgFile = "build.cfg"
	isrFile = "zisrnames.go"
)

const isrHeader = `// DO NOT EDIT THIS FILE. Generated by emgo.

package main

import _ "unsafe"

`

var workDir string

func die(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func dieErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func updateFromFile(cfg map[string]string) {
	f, err := os.Open(cfgFile)
	if err != nil {
		f, err = os.Open(filepath.Join(filepath.Dir(workDir), cfgFile))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return
			}
			dieErr(err)
		}
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	ln := 0
	for scanner.Scan() {
		ln++
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		i := strings.IndexByte(line, '=')
		if i < 0 {
			die("%s:%d: syntax error\n", f.Name(), ln)
		}
		name := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if _, ok := cfg[name]; !ok {
			die("%s:%d: unknown name \"%s\"\n", f.Name(), ln, name)
		}
		cfg[name] = val
	}
	dieErr(scanner.Err())
}

func gointerrupthandler() bool {
	files, err := os.ReadDir(".")
	dieErr(err)
	for _, f := range files {
		if f.Type()&fs.ModeType == 0 && filepath.Ext(f.Name()) == ".go" {
			content, err := os.ReadFile(f.Name())
			dieErr(err)
			if bytes.Contains(content, []byte("//go:interrupthandler")) {
				return true
			}
		}
	}
	return false
}

func noos(cmd *exec.Cmd, cfg map[string]string) {
	// Infer GOARCH, GOARM, GOTEXT from GOTARGET
	gotarget := cfg["GOTARGET"]
	if gotarget == "" {
		die("GOTARGET variable is not set\n")
	}
	def, ok := defaults[gotarget]
	if !ok {
		die("unknow GOTARGET: \"%s\"\n", gotarget)
	}
	cfg["GOARCH"] = def.GOARCH
	if cfg["GOARM"] == "" {
		cfg["GOARM"] = def.GOARM
	}

	dieErr(os.Setenv("GOOS", cfg["GOOS"]))
	dieErr(os.Setenv("GOARCH", cfg["GOARCH"]))
	dieErr(os.Setenv("GOARM", cfg["GOARM"]))

	if len(os.Args) < 2 || os.Args[1] != "build" {
		return
	}

	if cfg["GOTEXT"] == "" {
		cfg["GOTEXT"] = def.GOTEXT
	}

	// Check mandatory variables
	if cfg["GOTEXT"] == "" {
		die("GOTEXT variable is not set\n")
	}
	if cfg["GOMEM"] == "" {
		die("GOMEM variable is not set\n")
	}

	// Generate zisrnames.go
	if cfg["ISRNAMES"] == "" && !gointerrupthandler() {
		cfg["ISRNAMES"] = "no"
	}
	if cfg["ISRNAMES"] == "" {
		cfg["ISRNAMES"] = "github.com/embeddedgo/" + def.ISRNAMES
	}
	if path := cfg["ISRNAMES"]; path != "no" {
		ctx := &build.Default
		ctx.GOARCH = cfg["GOARCH"]
		ctx.GOOS = cfg["GOOS"]
		ctx.BuildTags = []string{cfg["GOTARGET"]}
		pkg, err := ctx.Import(path, "", 0)
		dieErr(err)
		eq := []byte{'='}
		buf := []byte(isrHeader)
		for _, name := range pkg.GoFiles {
			f, err := os.Open(filepath.Join(pkg.Dir, name))
			dieErr(err)
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := bytes.Fields(bytes.TrimSpace(scanner.Bytes()))
				if len(line) < 3 {
					continue
				}
				name, num := line[0], line[0][:0]
				if bytes.Equal(line[1], eq) {
					num = line[2]
				} else if len(line) >= 4 && bytes.Equal(line[2], eq) {
					num = line[3]
				}
				if len(num) == 0 {
					continue
				}
				buf = append(buf, "//go:linkname "...)
				buf = append(buf, name...)
				buf = append(buf, "_Handler IRQ"...)
				buf = append(buf, num...)
				buf = append(buf, "_Handler\n"...)
			}
			dieErr(scanner.Err())
			f.Close()
		}
		dieErr(os.WriteFile(isrFile, buf, 0666))
	}

	var tags, ldflags, o string
	flag.StringVar(&tags, "tags", "", "")
	flag.StringVar(&ldflags, "ldflags", "", "")
	flag.StringVar(&o, "o", "", "")
	flag.Parse()
	if tags != "" {
		tags += ","
	}
	tags += cfg["GOTARGET"]
	if ldflags != "" {
		tags += " "
	}
	ldflags += "-M " + cfg["GOMEM"]
	if cfg["GOTEXT"] != "-" {
		ldflags += " -T " + cfg["GOTEXT"]
	}
	if cfg["GOSTRIPFN"] != "" {
		ldflags += " -stripfn " + cfg["GOSTRIPFN"]
	}
	if o == "" {
		o = filepath.Base(workDir + ".elf")
	}
	cmd.Args = append(
		[]string{cmd.Args[0], "build", "-tags", tags, "-ldflags", ldflags, "-o", o},
		flag.Args()[1:]...,
	)
	if cmd.Run() != nil {
		os.Exit(1)
	}

	obj := strings.TrimSuffix(o, ".elf")
	os.Remove(isrFile)
	os.Remove(obj + ".hex")
	os.Remove(obj + "-settings.hex")
	os.Remove(obj + ".bin")

	if cfg["GOOUT"] != "" {
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
			cmd.Args = []string{objcopy, "-O", "ihex", o, obj + ".hex"}
		case "bin":
			cmd.Args = []string{objcopy, "-O", "binary", o, obj + ".bin"}
		default:
			die("unknown GOOUT: \"%s\"\n", cfg["GOOUT"])
		}
		if cmd.Run() != nil {
			os.Exit(1)
		}
	}

	os.Exit(0)
}

func main() {
	// Check if there is a local go tree next to the emgo executable.
	path, err := os.Executable()
	dieErr(err)
	path = filepath.Join(filepath.Dir(path), "go")
	_, err = os.Stat(filepath.Join(path, "VERSION"))

	var goCmd string
	if err == nil {
		// Use the local ./go/bin/go tool.
		goCmd = filepath.Join(path, "bin", "go")
	} else {
		// Otherwise use go tool from PATH if available.
		goCmd, err = exec.LookPath("go")
		dieErr(err)
	}

	cmd := &exec.Cmd{
		Path:   goCmd,
		Args:   os.Args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	// Initialize all known variables from environment.
	cfg := map[string]string{
		"GOOS":      os.Getenv("GOOS"),
		"GOTARGET":  os.Getenv("GOTARGET"),
		"GOARM":     os.Getenv("GOARM"),
		"GOTEXT":    os.Getenv("GOTEXT"),
		"GOMEM":     os.Getenv("GOMEM"),
		"GOOUT":     os.Getenv("GOOUT"),
		"GOSTRIPFN": os.Getenv("GOSTRIPFN"),
		"ISRNAMES":  os.Getenv("ISRNAMES"),
	}

	workDir, err = os.Getwd()
	dieErr(err)

	// Update variables from build.cfg file in current working directory or
	// its parent directory.
	updateFromFile(cfg)

	// GOOS defaults to noos if GOTARGET is set
	if cfg["GOOS"] == "" && cfg["GOTARGET"] != "" {
		cfg["GOOS"] = "noos"
	}

	if cfg["GOOS"] == "noos" {
		noos(cmd, cfg)
	}

	if cmd.Run() != nil {
		os.Exit(1)
	}
}

var defaults = map[string]struct{ GOARCH, GOARM, GOTEXT, ISRNAMES string }{
	"imxrt1060": {"thumb", "7d", "0x60002000", ""},
	"k210":      {"riscv64", "", "-", "kendryte/hal/irq"},
	"nrf52840":  {"thumb", "7", "", "nrf5/hal/irq"},
	"stm32f215": {"thumb", "7", "0x8000000", "stm32/hal/irq"},
	"stm32f407": {"thumb", "7", "0x8000000", "stm32/hal/irq"},
	"stm32f412": {"thumb", "7", "0x8000000", "stm32/hal/irq"},
	"stm32h7x3": {"thumb", "7d", "0x8000000", "stm32/hal/irq"},
	"stm32l4x6": {"thumb", "7", "0x8000000", "stm32/hal/irq"},
}
