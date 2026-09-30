.PHONY: build run test vet css docker

VERSION ?= $(shell git describe --tags --always 2>/dev/null | sed 's/^v//' || echo dev)
LDFLAGS := -X github.com/zachcurry13/recipebank/internal/version.Version=$(VERSION)

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/recipebank ./cmd/recipebank

run: build
	RECIPEBANK_DATA_DIR=./data RECIPEBANK_ADDR=127.0.0.1:8080 ./bin/recipebank

test:
	go test ./...

vet:
	go vet ./...

# Recompile Tailwind after changing classes in web/static (output is committed
# so `go build` needs no Node toolchain).
css:
	npx tailwindcss@3 -c tailwind.config.js -i web/tailwind.input.css -o web/static/css/app.css --minify

docker:
	docker build --build-arg VERSION=$(VERSION) -t recipebank:dev .
