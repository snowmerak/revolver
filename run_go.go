package main

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func runCommand(ctx context.Context, path string, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func RunGoApp(ctx context.Context, path string) error {
	sessionId := strings.ReplaceAll(uuid.New().String(), "-", "")

	buildFile := "app-" + sessionId

	log.Info().Msg("build new go app")
	if err := runCommand(ctx, path, "go", "build", "-o", buildFile, "."); err != nil {
		log.Error().Err(err).Msg("failed to run go app")
	}

	log.Info().Msg("run go app")
	if err := runCommand(ctx, path, "./"+buildFile); err != nil {
		log.Error().Err(err).Msg("failed to run go app")
	}

	log.Info().Msg("remove go app file")
	if err := os.Remove(buildFile); err != nil {
		log.Error().Err(err).Msg("failed to remove go app")
	}

	log.Info().Msg("go app stopped")

	return nil
}
