//go:build integration && !windows

package main

import (
	"os"
	"os/exec"
)

// SiapkanSinyal tidak melakukan apa-apa di Unix karena sinyal boleh dikirim
// langsung ke proses anak.
func SiapkanSinyal(cmd *exec.Cmd) {}

// KirimSinyalAndal mengirim SIGINT, yang oleh signal.Notify dipetakan ke
// os.Interrupt sehingga sama dengan Ctrl+Break di Windows.
func KirimSinyalAndal(p *os.Process) error {
	return p.Signal(os.Interrupt)
}
