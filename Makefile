.DEFAULT_GOAL := build

# Format Code
format:
	@echo "Formatting code:"
	go fmt ./...
.PHONY:format

# Check Code Style
# go install honnef.co/go/tools/cmd/staticcheck@latest
# go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
lint: format
	@echo "Linting code:"
	staticcheck ./...
	shadow ./...
	go vet ./...
.PHONY:lint

# Test Code
test: lint
	@echo "Testing code:"
	go test ./...
.PHONY:test

# Update Dependencies
dependencies:
	@echo "Updating dependencies:"
	go get -u ./...
	go mod tidy
.PHONY:dependencies

# Install/Update Tools
tools:
	@echo "Installing/updating tools:"
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
	go install github.com/swaggo/swag/cmd/swag@latest
.PHONY:tools

# Swaggo gin-swagger API Documentation
# go install github.com/swaggo/swag/cmd/swag@latest
docs:
	@echo "Generating API documentation:"
	swag init -g api.go -d cmd/api/ -o cmd/api/docs -ot go,json -pd -q
.PHONY:docs

# Build the command-line applications
build: docs
	@echo "Building API Lambda function for local use:"
	go build -tags lambda.norpc -ldflags "-X main.gitHash=`git rev-parse HEAD` -X main.gitOrigin=`git config --get remote.origin.url`" -o ./api ./cmd/api/*.go
	@echo "Building Operations Command for local use:"
	go build -ldflags "-X main.gitHash=`git rev-parse HEAD`" -o ./ops ./cmd/ops/*.go
.PHONY:build

# Build and package the API Lambda function for release
release: docs
	@echo "Building API Lambda function for release:"
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -ldflags "-X main.gitHash=`git rev-parse HEAD` -X main.gitOrigin=`git config --get remote.origin.url`" -o ./bootstrap ./cmd/api/*.go
	zip ./lambda.zip ./bootstrap
.PHONY:release

# Validate the CloudFormation template
validate:
	@echo "Validating CloudFormation template.yml ..."
	aws cloudformation validate-template \
		--region us-west-2 \
		--template-body file://template.yml
.PHONY:validate

# Package the CloudFormation template
package: release validate
	@echo "Packaging CloudFormation template.yml ..."
	test -f packaged-template.yml && rm packaged-template.yml || true
	aws cloudformation package \
		--region us-west-2 \
		--template-file template.yml \
		--s3-bucket versionary-lambdas \
		--output-template-file packaged-template.yml
.PHONY:package

# Deploy the packaged CloudFormation template
# make deploy env=[qa|staging|prod]
deploy: package
	@echo "Deploying packaged CloudFormation template ..."
	aws cloudformation deploy \
	  --region us-west-2 \
	  --template-file packaged-template.yml \
	  --stack-name versionary-api-$(env) \
	  --capabilities CAPABILITY_IAM \
	  --parameter-overrides ENV=$(env)
.PHONY:deploy
