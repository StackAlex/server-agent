package terminal

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/creack/pty/v2"
)

type PTY struct {
	File *os.File
	Cmd  *exec.Cmd
}

func Start(cmd *exec.Cmd) (*PTY, error) {
	file, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	return &PTY{
		File: file,
		Cmd:  cmd,
	}, nil
}

func (p *PTY) Write(data []byte) error {
	if p == nil || p.File == nil {
		return fmt.Errorf("PTY is not initialized")
	}

	written := 0

	for written < len(data) {
		n, err := p.File.Write(data[written:])
		if err != nil {
			return fmt.Errorf("failed to write to PTY: %w", err)
		}

		if n == 0 {
			return fmt.Errorf("failed to write to PTY: zero bytes written")
		}

		written += n
	}

	return nil
}

func (p *PTY) Read(buffer []byte) (int, error) {
	if p == nil || p.File == nil {
		return 0, fmt.Errorf("PTY is not initialized")
	}

	return p.File.Read(buffer)
}

func (p *PTY) Resize(cols, rows int) error {
	if p == nil || p.File == nil {
		return fmt.Errorf("PTY is not initialized")
	}

	if cols <= 0 || rows <= 0 {
		return fmt.Errorf("invalid terminal size: %dx%d", cols, rows)
	}

	return pty.Setsize(p.File, &pty.Winsize{
		Cols: uint16(cols),
		Rows: uint16(rows),
	})
}

func (p *PTY) Close() error {
	if p == nil {
		return nil
	}

	if p.File != nil {
		_ = p.File.Close()
	}

	if p.Cmd != nil && p.Cmd.Process != nil {
		_ = p.Cmd.Process.Kill()
	}

	return nil
}
