// Package backup dumps databases with pg_dump before destructive changes.
package backup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandRunner runs an external program and returns its combined output.
type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, env []string) ([]byte, error)
}

// ExecRunner runs commands with os/exec. env is added to the current
// environment.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.Bytes(), fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(out.String()))
	}
	return out.Bytes(), nil
}
