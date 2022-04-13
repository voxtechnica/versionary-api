# Versionary API

**Under Construction**

This project demonstrates a way of managing versioned entities in a database with a serverless architecture. It uses
the Go programming language with the following technologies:

* [AWS CloudFormation](https://aws.amazon.com/cloudformation/)
* [AWS DynamoDB](https://aws.amazon.com/dynamodb/)
* [AWS Lambda](https://aws.amazon.com/lambda/)
* [AWS API Gateway](https://aws.amazon.com/api-gateway/)
* [Gin Web Framework](https://gin-gonic.com/)

## Learning Resources

* [Go Programming Language](https://go.dev/) Home Page
* [Learning Go](https://learning.oreilly.com/library/view/learning-go/9781492077206/), by Jon Bodner (highly
  recommended for learning modern, idiomatic Go)
* [Effective Go](https://golang.org/doc/effective_go): a brief overview of the language
* [Go Language Standard Library](https://pkg.go.dev/std) package documentation
* [AWS Go Language SDK](https://aws.amazon.com/sdk-for-go/) Home Page
* [AWS Go Developer Guide](https://aws.github.io/aws-sdk-go-v2/docs/)
* [AWS Go API Reference](https://docs.aws.amazon.com/sdk-for-go/api/)
* [AWS Go Lambda Functions](https://docs.aws.amazon.com/lambda/latest/dg/lambda-golang.html)
* [AWS Go Lambda API Proxy](https://pkg.go.dev/github.com/awslabs/aws-lambda-go-api-proxy)
* [AWS DynamoDB Main Package](https://pkg.go.dev/github.com/aws/aws-sdk-go/service/dynamodb)
* [AWS DynamoDB Expression](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression)
* [AWS DynamoDB AttributeValue](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue)

## Dependencies

The following dependencies are required to build Versionary:

```bash
go get github.com/aws/aws-lambda-go/events
go get github.com/aws/aws-lambda-go/lambda
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/ssm
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
go get github.com/aws/aws-sdk-go-v2/service/dynamodb/types
go get github.com/awslabs/aws-lambda-go-api-proxy
go get github.com/gin-gonic/gin
go get github.com/voxtechnica/tuid-go
go get github.com/voxtechnica/versionary
go get golang.org/x/exp/slices
```

To update all the dependencies to their latest or minor patch releases, run the following command from the project root:

```bash
go get -u ./...
```

Once you've done this, you can test *all* the dependencies by running the following command from the project root:

```bash
go test all
```

That can take a while. If you prefer, you can test just this project:

```bash
go test ./...
```

## Major Infrastructure Elements

* GitHub code repository
* CloudFormation template
* API Gateway HTTP Proxy
* Versionary API Lambda Function
* DynamoDB Tables (created using an operations command)

**Release process**:

A CloudFormation template ("infrastructure as code") is used to create and manage most of the infrastructure elements.
Exceptions include DynamoDB tables (created using an operations command), SSL Certificates, Route 53 DNS records, etc.

There are many ways to manage the code running in different operating environments. Once you have code ready for release
to a particular environment, you can use the following steps to deploy the code to that environment:

1. Ensure that you have the desired git branch checked out locally, and that all tests pass.
2. Run `make release` to build deployment artifacts.
3. Run `cmd/ops/operations -env <env> -action create-tables` to create any missing DynamoDB tables in the environment.
4. Run `./deploy.sh <env>` to package and deploy the CloudFormation template (`template.yml`).
5. Test the updated code running in the environment.

## Initial Setup

1. Install software and configure your workstation, as indicated below.
2. Create an S3 bucket for Lambda functions (e.g. `versionary-lambdas`), used by `deploy.sh`.
3. Provision and configure services for sending messages, as described for
   the [Message Service](./doc/MessageServiceSetup.md).
4. Create a bootstrap admin bearer token, as described for the [User Service](./doc/UserServiceSetup.md).

## Developer Workstation Setup

1. Install the [AWS Command Line Interface](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html).
2. Configure your AWS IAM credentials using `aws configure`, supplying your access key, secret key, and default region.
3. Install the [Go Language](https://golang.org/doc/install).
4. Install an IDE, such as [VSCode](https://code.visualstudio.com/), [GoLand](https://www.jetbrains.com/go/),
   or [IntelliJ IDEA](http://www.jetbrains.com/idea/). VSCode with the Go plugin is excellent value (free!), but the
   JetBrains IDEs (GoLand or IntelliJ IDEA with the Go plugin) offer more. In addition to outstanding refactoring
   support, they actually teach you to write better code as you use the tools.
