# revolver

Revolver is a simple CLI tool for hot reloading your Go applications.  
Revolver detects changes in your project files and restarts your application automatically.

## Installation

```bash
go install github.com/snowmerak/revolver@latest
```

## Usage

```bash
revolver <project-root-path> <main-package-path> <extensions>...
```

## Example

If you have a project structure like this:

```bash
.
├── cmd
│   └── server
│       └── main.go
└── go.mod
```

You can run the following command to hot reload your server:

```bash
revolver . cmd/server
```

If your server is using `.go` files and `.html` files, you can run the following command:

```bash
revolver . cmd/server go html
```

But `.go` is the default extension, '.go' is not necessary:

```bash
revolver . cmd/server html
```
