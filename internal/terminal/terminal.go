package terminal

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/google/uuid"
)

type Terminal struct {
	ID  string
	Cmd *exec.Cmd
	PTY *PTY

	writeMu sync.Mutex
}

func New(cols, rows int) (*Terminal, error) {
	cmd := exec.Command("/bin/bash", "-l")

	// Запускаем терминал из домашней директории пользователя
	cmd.Dir = "/home/webadmin"

	ptmx, err := Start(cmd)
	if err != nil {
		return nil, err
	}

	if err := ptmx.Resize(cols, rows); err != nil {
		_ = ptmx.Close()

		return nil, fmt.Errorf("failed to resize terminal: %w", err)
	}

	return &Terminal{
		ID:  uuid.NewString(),
		Cmd: cmd,
		PTY: ptmx,
	}, nil
}

func (t *Terminal) Write(data []byte) error {
	if t == nil || t.PTY == nil {
		return fmt.Errorf("terminal is not initialized")
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	return t.PTY.Write(data)
}

func (t *Terminal) Read(buffer []byte) (int, error) {
	if t == nil || t.PTY == nil {
		return 0, fmt.Errorf("terminal is not initialized")
	}

	return t.PTY.Read(buffer)
}

func (t *Terminal) Resize(cols, rows int) error {
	if t == nil || t.PTY == nil {
		return fmt.Errorf("terminal is not initialized")
	}

	return t.PTY.Resize(cols, rows)
}

func (t *Terminal) Close() error {
	if t == nil || t.PTY == nil {
		return nil
	}

	return t.PTY.Close()
}
