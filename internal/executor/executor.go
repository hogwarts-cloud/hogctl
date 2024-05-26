package executor

import (
	"fmt"
	"os/exec"
)

type Executor struct {
}

func (e *Executor) Execute(command string, args []string) error {
	cmd := exec.Command(command, args...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run cmd: %w", err)
	}

	return nil
}

func New() *Executor {
	return &Executor{}
}
