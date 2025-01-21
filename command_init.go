package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const CommandInit = "init"

var CommandInitNotFoundFileNameError = errors.New("not found file name")

func CommandInitFunc(args []string) error {
	if len(args) != 1 {
		fmt.Printf("Usage: %s %s <filename>\n", os.Args[0], CommandInit)
		return CommandInitNotFoundFileNameError
	}

	filename := args[0]
	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	cfg := RevolverConfig{
		LogLevel:                LogLevelInfo,
		ProjectRootFolder:       ".",
		ExecutablePackageFolder: "cmd/revolver",
		Ports: []RevolverPortConfig{
			{
				Port: 8080,
				Name: "http",
				Env:  "HTTP_PORT",
			},
		},
		Scripts: RevolverScriptConfig{
			Preload: "go build -o app .",
			Run:     "./app",
			CleanUp: "rm app",
		},
		ObservingExts: []string{".go", ".mod", ".sum"},
	}

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(cfg); err != nil {
		return err
	}
	encoder.Close()

	return nil
}
