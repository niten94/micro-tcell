// +build darwin

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

// The Darwin system is *almost* a real BSD system, but it suffers from
// a brain damaged TTY driver.  This TTY driver does not actually
// wake up in poll() or similar calls, which means that we cannot reliably
// shut down the terminal without resorting to obscene custom C code
// and a dedicated poller thread.
//
// So instead, we do a best effort, and simply try to do the close in the
// background.  Probably this will cause a leak of two goroutines and
// maybe also the file descriptor, meaning that applications on Darwin
// can't reinitialize the screen, but that's probably a very rare behavior,
// and accepting that is the best of some very poor alternative options.
//
// Maybe someday Apple will fix there tty driver, but its been broken for
// a long time (probably forever) so holding one's breath is contraindicated.
//
// NOTE: In this fork of tcell, we fix this issue by using the library
// zyedidia/poller to properly interface with the tty such that when we
// close it, it actually closes

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/zyedidia/poller"

	"golang.org/x/term"
)

func (t *tScreen) termioInit() error {
	var e error
	var state *term.State

	if t.in, e = poller.Open("/dev/tty", poller.O_RO); e != nil {
		goto failed
	}
	if t.out, e = poller.Open("/dev/tty", poller.O_WO); e != nil {
		goto failed
	}

	state, e = term.MakeRaw(int(t.out.(*poller.FD).Sysfd()))
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
		t.in.(*poller.FD).Close()
	}
	if t.out != nil {
		t.out.(*poller.FD).Close()
	}
	return e
}

func (t *tScreen) termioFini() {

	signal.Stop(t.sigwinch)

	<-t.indoneq

	if t.out != nil && t.saved != nil {
		term.Restore(int(t.out.(*poller.FD).Sysfd()), t.saved)
		t.out.(*poller.FD).Close()
	}

	if t.in != nil {
		t.in.(*poller.FD).Close()
	}
}

func (t *tScreen) getWinSize() (int, int, error) {
	cols, rows, err := term.GetSize(int(t.out.(*poller.FD).Sysfd()))
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
