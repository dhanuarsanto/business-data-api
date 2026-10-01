//go:build integration && windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	proGenerateCtrlEvent         = kernel32.NewProc("GenerateConsoleCtrlEvent")
	ctrlBreakEvent       uintptr = 1
)

// SiapkanSinyal meletakkan anak proses di kelompok proses baru supaya
// GenerateConsoleCtrlEvent terarah hanya ke anak, bukan ke proses uji.
func SiapkanSinyal(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// KirimSinyalAndal meniru Ctrl+Break. taskkill /F memakai TerminateProcess yang
// tidak bisa ditangkap signal.Notify, sehingga jalur shutdown bersih di main()
// tidak akan pernah berjalan. Nilai balik, bukan error dari GetLastError, yang
// menentukan berhasil atau tidak: GetLastError selalu mengembalikan kode sisa.
func KirimSinyalAndal(p *os.Process) error {
	rv, _, _ := proGenerateCtrlEvent.Call(ctrlBreakEvent, uintptr(p.Pid), 0)
	if rv == 0 {
		return fmt.Errorf("GenerateConsoleCtrlEvent gagal untuk pid %d", p.Pid)
	}
	return nil
}
