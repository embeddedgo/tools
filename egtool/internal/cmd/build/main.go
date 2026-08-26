// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/embeddedgo/tools/egtool/internal/util"
)

const (
	DescrBuild = "run `go build` with GOENV set to the found go.env file"
	DescrRun   = "run `go run` with GOENV set to the found go.env file"
	DescrTest  = "run `go test` with GOENV set to the found go.env file"
)

const help = `
This command looks for the go.env file up the current module directory tree and
sets the GOENV enviroment variable to it. Next it runs the go %s command with
out any arguments. It is inteneded for simple use cases when you build the code
in the current directory and all required build options are provided by the
go.env file.
`

func Main(cmd string, args []string) {
	gotool, err := exec.LookPath("go")
	util.FatalErr("", err)
	c := &exec.Cmd{
		Path:   gotool,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if cmd == "build" {
		c.Args = append([]string{gotool, cmd}, args...)
	} else {
		c.Args = append(
			[]string{gotool, cmd, "-exec", os.Args[0] + " exec"},
			args...,
		)
	}
	goenvPath := util.SetGOENV(false)
	if goenvPath != "" && (cmd == "build" || cmd == "test") && len(args) != 0 {
		fmt.Println("GOENV set to", goenvPath)
	}
	err = c.Run()
	if err == nil {
		return
	}
	if ee, ok := err.(*exec.ExitError); ok {
		os.Exit(ee.ProcessState.ExitCode())
	}
	util.FatalErr("", err)
}
