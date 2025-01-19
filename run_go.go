package main

import (
	"context"
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
		log.Error().Err(err).Msg("failed to run app")
	}

	cmd.Process.Wait()
	log.Info().Msg("app stopped")

	return nil
}
