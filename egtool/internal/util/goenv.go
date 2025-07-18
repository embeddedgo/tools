// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

const goenvName = "go.env"

// SetGOENV tries to find the go.env file and set GOENV to it.
func SetGOENV() {
	if os.Getenv("GOOS") == "noos" && os.Getenv("GOENV") != "" {
		// GOENV is set by user manualy to be used together with GOOS=noos
		return
	}
	wd, err := os.Getwd()
	if err != nil {
		FatalErr("", err)
	}
	var goenvPath string
	for {
		goenvPath = filepath.Join(wd, goenvName)
		fi, err := os.Stat(goenvPath)
		if err == nil {
			if !fi.Mode().IsRegular() {
				Fatal("%s is not a regular file", goenvPath)
			}
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			FatalErr("", err)
		}
		_, err = os.Stat(filepath.Join(wd, "go.mod"))
		if err == nil {
			return // found go.mod but no goenvName, stop here
		}
		if !errors.Is(err, fs.ErrNotExist) {
			FatalErr("", err)
		}
		wd = filepath.Dir(wd)
		if wd == "/" || wd == "." { // FIXME: windows?
			return
		}
	}
	FatalErr("", os.Setenv("GOENV", goenvPath))
}
