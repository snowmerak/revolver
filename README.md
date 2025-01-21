# revolver

Revolver is a simple CLI tool for hot reloading your Go applications.  
Revolver detects changes in your project files and restarts your application automatically.

## Installation

```bash
go install github.com/snowmerak/revolver@latest
```

## Usage

```bash
revolver <command> <config-file>
```

### Init

```bash
revolver init dev.yaml
```

This command will create a `dev.yaml` file in the current directory with the following content:

```yaml
log_level: info
root: .
exec: .
ports:
  - port: 8080
    name: http
    env: PORT
scripts:
  preload: go build -o app .
  run: ./app
  cleanup: rm app
exts:
  - .go
```

### Watch

```bash
revolver watch dev.yaml
```

This command will start watching the files in the current directory and restart the application when a change is detected.

## Example

If you have a project structure like this:

```bash
.
├── cmd
│   └── server
│       └── main.go
└── go.mod
```

You must edit the `dev.yaml` file to match your project structure:

```yaml
log_level: info
root: .
exec: ./cmd/server
ports:
  - port: 8080
    name: http
    env: PORT
scripts:
  preload: go build -o app .
  run: ./app
  cleanup: rm app
exts:
  - .go
  - .mod
  - .sum
```

Then you can run the following command:

```
revolver watch dev.yaml
```
