// +build aix dragonfly freebsd linux netbsd openbsd solaris zos

// Copyright 2019 The TCell Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tcell

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"golang.org/x/term"
)

func (t *tScreen) termioInit() error {
	var e error
	var state *term.State

	if t.in, e = os.OpenFile("/dev/tty", os.O_RDONLY, 0); e != nil {
		goto failed
	}
	if t.out, e = os.OpenFile("/dev/tty", os.O_WRONLY, 0); e != nil {
		goto failed
	}

	state, e = term.MakeRaw(int(t.out.(*os.File).Fd()))
	if e != nil {
		goto failed
	}

	t.saved = state

	signal.Notify(t.sigwinch, syscall.SIGWINCH)

	if w, h, e := t.getWinSize(); e == nil && w != 0 && h != 0 {
		t.cells.Resize(w, h)
	}

	return nil

failed:
	if t.in != nil {
		t.in.(*os.File).Close()
	}
	if t.out != nil {
		t.out.(*os.File).Close()
	}
	return e
}

func (t *tScreen) termioFini() {

	signal.Stop(t.sigwinch)

	<-t.indoneq

	if t.out != nil && t.saved != nil {
		term.Restore(int(t.out.(*os.File).Fd()), t.saved)
		t.out.(*os.File).Close()
	}

	if t.in != nil {
		t.in.(*os.File).Close()
	}
}

func (t *tScreen) getWinSize() (int, int, error) {
	cols, rows, err := term.GetSize(int(t.out.(*os.File).Fd()))
	if err != nil {
		return -1, -1, err
	}
	if cols == 0 {
		colsEnv := os.Getenv("COLUMNS")
		if colsEnv != "" {
			if cols, err = strconv.Atoi(colsEnv); err != nil {
				return -1, -1, err
			}
		} else {
			cols = t.ti.Columns
		}
	}
	if rows == 0 {
		rowsEnv := os.Getenv("LINES")
		if rowsEnv != "" {
			if rows, err = strconv.Atoi(rowsEnv); err != nil {
				return -1, -1, err
			}
		} else {
			rows = t.ti.Lines
		}
	}
	return cols, rows, nil
}

func (t *tScreen) Beep() error {
	t.writeString(string(byte(7)))
	return nil
}
