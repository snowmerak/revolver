package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func RunGoApp(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "go", "run", ".")
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if !errors.Is(err, context.Canceled) {
			return context.Canceled
		}

		return fmt.Errorf("failed to run go app: %w", err)
	}

	return nil
}
