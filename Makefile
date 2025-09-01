.PHONY: build clean

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

comp:
	@echo "🔧 Generating ZSH completion..."
	@mkdir -p ~/dotfiles/.config/zsh/completion
	@go run cmd/$(BINARY_NAME)/main.go --no-update-check completion zsh > ~/dotfiles/.config/zsh/completion/_$(BINARY_NAME)
	@chmod +x ~/dotfiles/.config/zsh/completion/_$(BINARY_NAME)
	@chmod 644 ~/dotfiles/.config/zsh/completion/_$(BINARY_NAME)
	@echo "✅ ZSH completion generated in ~/dotfiles/.config/zsh/completion/_$(BINARY_NAME)"