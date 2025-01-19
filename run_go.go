package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

func RunGoApp(ctx context.Context, path string) error {
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	context.AfterFunc(ctx, func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Process.Signal(os.Kill)
		cmd.Process.Kill()
	})

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stopped app: %w", err)
	}

	cmd.Process.Wait()
	log.Info().Msg("app stopped")

	return nil
}
