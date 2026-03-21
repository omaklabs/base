.PHONY: dev build css css-watch generate lint test clean help

# Full build: generate code, compile CSS, compile Go binary
build: generate css
	go build -o ./app ./cmd/app

# Build and start the dev server
dev: build
	./app serve

# Compile Tailwind CSS for production (dev uses browser CDN instead)
css:
	tailwindcss -i assets/css/app.css -o assets/css/app.compiled.css --minify

# Watch and recompile Tailwind CSS for production
css-watch:
	tailwindcss -i assets/css/app.css -o assets/css/app.compiled.css --watch

# Run all code generators (templ + sqlc)
generate:
	templ generate
	sqlc generate

lint:
	golangci-lint run

test:
	go test ./...

clean:
	rm -f ./app

help:
	@echo "make build       Full build (generate + css + go build)"
	@echo "make dev         Build and start the server"
	@echo "make css         Compile Tailwind CSS for production"
	@echo "make css-watch   Watch and recompile Tailwind CSS for production"
	@echo "make generate    Run templ generate + sqlc generate"
	@echo "make lint        Run golangci-lint"
	@echo "make test        Run all tests"
	@echo "make clean       Remove build artifacts"
