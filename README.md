# revolver

Revolver is a simple CLI tool for live reloading your Go applications.  
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

### ReverseProxy

Revolver can also act as a tcp reverse proxy for your application.  
This is useful when deploying your application with zero downtime.  
But revolver is using `proxyprotocol v2` to pass the client's IP address to the application.  
You must implement the `proxyprotocol v2` in your application.

But your application written in Go can use the `github.com/snowmerak/revolver/listener` package to use the `proxyprotocol v2`.

```bash
go get github.com/snowmerak/revolver
```

```go
package main

import (
	"github.com/snowmerak/revolver/listener"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
        port = "8080"
	}
	
    ln, err := listener.New("0.0.0.0:" + port)
	if err != nil {
        log.Fatal(err)
    }
	
	...
}
````

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

## Dockerfile

You can also use revolver in a Dockerfile:

```Dockerfile
FROM docker.io/golang:latest

RUN go install github.com/snowmerak/revolver@latest

WORKDIR /app

COPY . .
# or MOUNT your project files

CMD ["revolver", "watch", "dev.yaml"]
```

```yaml
version: '3.8'
service:
    app:
        build:
            context:
            dockerfile: dev.Dockerfile
        ports:
        - 8080:8080
        volumes:
        - .:/app
```
