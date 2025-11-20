.PHONY: build clean lint lintf test cov comp buildt

BINARY_NAME=keg

build:
	@go build -ldflags "-s -w" -o ~/.local/bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "✅ Build complete. Executable is located at ~/.local/bin/$(BINARY_NAME)"

clean:
	@echo "🧹 Cleaning up..."
	@rm ~/.local/bin/$(BINARY_NAME)
	@echo "✅ Cleaned up. Executable removed from ~/.local/bin/$(BINARY_NAME)"

lint:
	@echo "🔍 Running linters..."
	@golangci-lint run --config .golangci.yml

lintf:
	@echo "🔍 Running linters..."
	@golangci-lint run --config .golangci.yml --fix

test:
	@echo "🧪 Running tests..."
	@go test -count=1 ./...

cov:
	@echo "🧪 Running tests with coverage..."
	@go test ./... -covermode=atomic -coverprofile=coverage.out
	@go tool cover -func=coverage.out | tail -n1

vuln:
	@echo "🔒 Checking for vulnerabilities..."
	@govulncheck ./...


comp:
	@echo "🔧 Generating ZSH completion..."
	@mkdir -p ~/dotfiles/.config/zsh/completion
	@go run cmd/$(BINARY_NAME)/main.go --no-update-check completion zsh > ~/dotfiles/.config/zsh/completion/_$(BINARY_NAME)
	@chmod +x ~/dotfiles/.config/zsh/completion/_$(BINARY_NAME)
	@chmod 644 ~/dotfiles/.config/zsh/completion/_$(BINARY_NAME)
	@echo "✅ ZSH completion generated in ~/dotfiles/.config/zsh/completion/_$(BINARY_NAME)"



VERSION = 0.2.1
COMMIT  = $(shell git rev-parse --short HEAD)
DATE     = $(shell date -u -d "2 weeks ago" +%Y-%m-%dT%H:%M:%SZ)
GOVERSION = $(shell go version | awk '{print $$3}')

LDFLAGS = -X github.com/MrSnakeDoc/keg/internal/checker.Version=$(VERSION) \
           -X github.com/MrSnakeDoc/keg/internal/checker.Commit=$(COMMIT) \
           -X github.com/MrSnakeDoc/keg/internal/checker.Date=$(DATE) \
		   -X github.com/MrSnakeDoc/keg/internal/checker.GoVersion=$(GOVERSION)

buildt:
	go build -ldflags "$(LDFLAGS)" -o bin/keg cmd/keg/main.go