.DEFAULT_GOAL := build

# Format Code
format:
	@echo "Formatting code:"
	go fmt ./...
.PHONY:format

# Check Code Style
# go install honnef.co/go/tools/cmd/staticcheck@latest
lint: format
	@echo "Linting code:"
	staticcheck ./...
	go vet ./...
.PHONY:lint

# Build the command-line applications
build:
	@echo "Building API Lambda function:"
	go build -o ./cmd/api/api ./cmd/api/api.go
	@echo "Building Operations Command for local use ..."
	go build -o ./cmd/ops/operations ./cmd/ops/operations.go
.PHONY:build

release:
	@echo "Building API Lambda function for release:"
	GOOS=linux GOARCH=amd64 go build -o ./cmd/api/api ./cmd/api/api.go
	zip ./cmd/api/lambda.zip ./cmd/api/api
.PHONY:release
