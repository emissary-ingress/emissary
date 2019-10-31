// +build flake

package main

import (
	"syscall"
	"testing"
)

// We should really do something sane in each case:
//  - startup with good, switch to bad
//  - startup with bad, switch to good
//  - startup with good, switch to alt
// Currently we only cover good to alternative good, I haven't figured
// out what makes sense in the other cases

func TestHUPGood2Alt(t *testing.T) {
	if noDocker != nil {
		t.Skip(noDocker)
	}
	gotHere := false
	writeGoodFile(HupConfig)
	withInterrupt(t, hup, func() {
		if poll(t, "http://httptarget", "HTTPTEST") {
			writeAltFile(HupConfig)
			err := hup.Process.Signal(syscall.SIGHUP)
			if err != nil {
				t.Errorf("error sending signal: %v", err)
				return
			}
			if poll(t, "http://httptarget", "ALT") {
				gotHere = true
				return
			}
		}
	})
	if !gotHere {
		t.Errorf("didn't get there")
	}
}
