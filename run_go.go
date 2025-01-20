package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

func runCommand(ctx context.Context, env []string, path string, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func ParseCommand(content string) ([]string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.Comma = ' '
	reader.LazyQuotes = true

	record, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error parsing content: %w", err)
	}

	return record, nil
}

func RunCommandSet(ctx context.Context, env []string, path string, script RevolverScriptConfig) error {
	osEnv := os.Environ()
	cmdEnv := make([]string, len(osEnv)+len(env))
	copy(cmdEnv, osEnv)
	copy(cmdEnv[len(osEnv):], env)

	preloadCommands, err := ParseCommand(script.Preload)
	if err != nil {
		return fmt.Errorf("failed to parse preload commands: %w", err)
	}

	if err := runCommand(ctx, cmdEnv, path, preloadCommands[0], preloadCommands[1:]...); err != nil {
		return fmt.Errorf("failed to run preload command: %w", err)
	}

	runCommands, err := ParseCommand(script.Run)
	if err != nil {
		return fmt.Errorf("failed to parse run commands: %w", err)
	}

	if err := runCommand(ctx, cmdEnv, path, runCommands[0], runCommands[1:]...); err != nil {
		return fmt.Errorf("failed to run run command: %w", err)
	}

	context.AfterFunc(ctx, func() {
		stopCommands, err := ParseCommand(script.CleanUp)
		if err != nil {
			log.Error().Err(err).Msg("failed to parse cleanup commands")
			return
		}

		if err := runCommand(ctx, cmdEnv, path, stopCommands[0], stopCommands[1:]...); err != nil {
			log.Error().Err(err).Msg("failed to run cleanup command")
			return
		}
	})

	return nil
}
