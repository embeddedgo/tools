// Copyright 2022 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"bytes"
	"errors"
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

func updateCfgFromFile(cfg map[string]string) {
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
		if _, ok := cfg[name]; !ok && name != "DISABLE" {
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

func noosBuild(cmd *exec.Cmd, cfg map[string]string) {
	// Infer GOARCH, GOARM, GOTEXT from GOTARGET
	gotarget := cfg["GOTARGET"]
	if gotarget == "" {
		die("GOTARGET variable is not set\n")
	}
	def, ok := defaults[gotarget]
	if !ok {
		die("unknow GOTARGET: \"%s\"\n", gotarget)
	}
	if cfg["GOARCH"] == "" {
		cfg["GOARCH"] = def.GOARCH
	}
	if cfg["GOARM"] == "" {
		cfg["GOARM"] = def.GOARM
	}

	dieErr(os.Setenv("GOOS", cfg["GOOS"]))
	dieErr(os.Setenv("GOARCH", cfg["GOARCH"]))
	dieErr(os.Setenv("GOARM", cfg["GOARM"]))

	// Check mandatory variables
	if cfg["GOTEXT"] == "" {
		switch cfg["GOTARGET"] {
		case "k210":
			// GOTEXT not used
		case "noostest":
			cfg["GOTEXT"] = noostest[cfg["GOARCH"]].GOTEXT // "" if not used
		default:
			die("GOTEXT variable is not set\n")
		}
	}
	if cfg["GOMEM"] == "" {
		if cfg["GOTARGET"] == "noostest" {
			cfg["GOMEM"] = noostest[cfg["GOARCH"]].GOMEM
		}
		if cfg["GOMEM"] == "" {
			die("GOMEM variable is not set\n")
		}
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

	// Get flags shared with the go command.
	var (
		tags, ldflags, out string
		cut                *string
		args               []string
	)
	for _, a := range os.Args[2:] {
		if cut != nil {
			*cut = a
			cut = nil
			continue
		}
		switch a {
		case "-tags":
			cut = &tags
		case "-ldflags":
			cut = &ldflags
		case "-o":
			cut = &out
		default:
			args = append(args, a)
		}
	}
	// Append emgo configuration to the flags.
	if tags != "" {
		tags += ","
	}
	tags += cfg["GOTARGET"]
	if ldflags != "" {
		ldflags += " "
	}
	ldflags += "-M " + cfg["GOMEM"]
	if cfg["GOTEXT"] != "" {
		ldflags += " -F " + cfg["GOTEXT"]
	}
	if cfg["GOSTRIPFN"] != "" {
		ldflags += " -stripfn " + cfg["GOSTRIPFN"]
	}
	if out == "" {
		out = filepath.Base(workDir) + ".elf"
	}
	cmd.Args = []string{cmd.Args[0], os.Args[1], "-tags", tags, "-ldflags", ldflags}
	if os.Args[1] == "test" {
		cmd.Args = append(cmd.Args, "-exec", "emgo")
	} else {
		cmd.Args = append(cmd.Args, "-o", out)
	}
	cmd.Args = append(cmd.Args, args...)
	if cmd.Run() != nil {
		os.Exit(1)
	}

	obj := strings.TrimSuffix(out, ".elf")
	os.Remove(isrFile)
	os.Remove(obj + ".hex")
	os.Remove(obj + "-settings.hex")
	os.Remove(obj + ".bin")

	if cfg["GOOUT"] != "" {
		objcopy(out, obj, cfg["GOOUT"], cfg["GOINCBIN"])
	}

	os.Exit(0)
}

func cfgFromEnv() map[string]string {
	return map[string]string{
		"GOOS":      os.Getenv("GOOS"),
		"GOARCH":    os.Getenv("GOARCH"),
		"GOTARGET":  os.Getenv("GOTARGET"),
		"GOARM":     os.Getenv("GOARM"),
		"GOTEXT":    os.Getenv("GOTEXT"),
		"GOMEM":     os.Getenv("GOMEM"),
		"GOOUT":     os.Getenv("GOOUT"),
		"GOINCBIN":  os.Getenv("GOINCBIN"),
		"GOSTRIPFN": os.Getenv("GOSTRIPFN"),
		"ISRNAMES":  os.Getenv("ISRNAMES"),
	}
}

func main() {
	if len(os.Args) >= 2 {
		if a1 := os.Args[1]; strings.HasSuffix(a1, ".test") || strings.HasSuffix(a1, ".elf") {
			// Use an emulator to run ELF binary.
			os.Exit(runELF())
		}
	}

	// Check if there is a local go tree next to the emgo executable.
	path, err := os.Executable()
	dieErr(err)
	path = filepath.Join(filepath.Dir(path), "go")
	_, err = os.Stat(filepath.Join(path, "VERSION"))

	var goCmd string
	if err == nil {
		// Use the local ./go toolchain.
		goCmd = filepath.Join(path, "bin", "go")
		// Override GOROOT
		dieErr(os.Setenv("GOROOT", path))
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

	workDir, err = os.Getwd()
	dieErr(err)

	// Initialize all known variables from environment.
	cfg := cfgFromEnv()

	// Update variables from build.cfg file in the current working directory or
	// its parent directory.
	updateCfgFromFile(cfg)

	if reason, ok := cfg["DISABLE"]; ok {
		cfg = cfgFromEnv()
		if cfg["GOTARGET"] == "" {
			fmt.Fprintln(os.Stderr, "The build.cfg file is disabled.")
			if reason != "" {
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, reason)
			}
			os.Exit(1)
		}
	}

	// GOOS defaults to noos if GOTARGET is set
	if cfg["GOOS"] == "" && cfg["GOTARGET"] != "" {
		cfg["GOOS"] = "noos"
	}

	if len(os.Args) >= 2 && cfg["GOOS"] == "noos" {
		switch os.Args[1] {
		case "build", "test":
			noosBuild(cmd, cfg)
		}
	}

	if cmd.Run() != nil {
		os.Exit(1)
	}
}

var defaults = map[string]struct{ GOARCH, GOARM, ISRNAMES string }{
	"imxrt1060": {"thumb", "7,hardfloat", "imxrt/hal/irq"},
	"k210":      {"riscv64", "", "kendryte/hal/irq"},
	"n64":       {"mips64", "", ""},
	"noostest":  {},
	"nrf52840":  {"thumb", "7,softfloat", "nrf5/hal/irq"},
	"stm32f215": {"thumb", "7,softfloat", "stm32/hal/irq"},
	"stm32f407": {"thumb", "7,softfloat", "stm32/hal/irq"},
	"stm32f412": {"thumb", "7,softfloat", "stm32/hal/irq"},
	"stm32h7x3": {"thumb", "7,hardfloat", "stm32/hal/irq"},
	"stm32l4x6": {"thumb", "7,softfloat", "stm32/hal/irq"},
}

var noostest = map[string]struct{ GOMEM, GOTEXT string }{
	"thumb":   {"0x60000000:16M,0x20000000:4M", "0x00000000:4M"},
	"riscv64": {"0x80000000:32M", ""},
}
