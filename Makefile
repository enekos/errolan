BINARY := errolan
PKG    := ./cmd/errolan

.PHONY: build run dev test vet fmt tidy clean

build:
	go build -o $(BINARY) $(PKG)

run: build
	./$(BINARY)

dev:
	@command -v air >/dev/null || { echo "air not installed: go install github.com/air-verse/air@latest"; exit 1; }
	@if [ -f .env ]; then set -a && . ./.env && set +a && air; else air; fi

test:
	go test ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BINARY) tmp build-errors.log
