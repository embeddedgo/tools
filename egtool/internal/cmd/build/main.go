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

const Descr = "run `go build` with GOENV set to the found go.env file"

const help = `
This command looks for the go.env file up the current module directory tree and
sets the GOENV enviroment variable to it. Next it runs the go build command with
out any arguments. It is inteneded for simple use cases when you build the code
in the current directory and all required build options are provided by the
go.env file.
`

func Main(cmd string, args []string) {
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, help)
		os.Exit(1)
	}
	util.SetGOENV(true)
	goCmd, err := exec.LookPath("go")
	util.FatalErr("", err)
	c := &exec.Cmd{
		Path:   goCmd,
		Args:   []string{goCmd, cmd},
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
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
