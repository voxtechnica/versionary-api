# Versionary API

This project is **Under Construction**.

This project demonstrates a way to manage versioned entities in a database with a serverless architecture.
It uses the Go programming language with the following technologies:

* [AWS CloudFormation](https://aws.amazon.com/cloudformation/)
* [AWS DynamoDB](https://aws.amazon.com/dynamodb/)
* [AWS Lambda](https://aws.amazon.com/lambda/)
* [AWS API Gateway](https://aws.amazon.com/api-gateway/)
* [AWS SSM Parameter Store](https://aws.amazon.com/systems-manager/features/#Parameter_Store)
* [AWS Simple Email Service (SES)](https://aws.amazon.com/ses/)
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

The following dependencies are required to build the Versionary API:

```bash
go get github.com/aws/aws-lambda-go/events
go get github.com/aws/aws-lambda-go/lambda
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/dynamodb
go get github.com/aws/aws-sdk-go-v2/service/dynamodb/types
go get github.com/aws/aws-sdk-go-v2/service/ses
go get github.com/aws/aws-sdk-go-v2/service/ssm
go get github.com/awslabs/aws-lambda-go-api-proxy
go get github.com/gin-gonic/gin
go get github.com/spf13/cobra
go get github.com/stretchr/testify
go get github.com/swaggo/files
go get github.com/swaggo/gin-swagger
go get github.com/voxtechnica/tuid-go
go get github.com/voxtechnica/user-agent
go get github.com/voxtechnica/versionary
go get golang.org/x/exp/slices
```

To update all the dependencies to their latest or minor patch releases, run the following command from the project root:

```bash
make dependencies
```

Once you've done this, you can test *all* the dependencies by running the following command from the project root:

```bash
go test all
```

That can take a while. If you prefer, you can test just this project:

```bash
make test
```

## Developer Workstation Setup

1. Install the [AWS Command Line Interface](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html).
2. Configure your AWS IAM credentials using `aws configure`, supplying your access key, secret key, and default region.
3. Install the [Go Language](https://golang.org/doc/install), and set up
   a [GOPATH environment variable](https://github.com/golang/go/wiki/SettingGOPATH).
4. Install an IDE, such as [VSCode](https://code.visualstudio.com/), [GoLand](https://www.jetbrains.com/go/),
   or [IntelliJ IDEA](http://www.jetbrains.com/idea/). VSCode with the Go plugin is excellent value (free!), and is
   probably the best choice for most developers. The JetBrains IDEs have better support for refactoring, and may be
   worth the cost if you're a professional developer.
5. Install some Go tools used in the [Makefile](Makefile) in your GOPATH bin folder:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
go install github.com/swaggo/swag/cmd/swag@latest
```

## Initial Environment Setup

1. Create an [S3 bucket](https://s3.console.aws.amazon.com/s3/buckets?region=us-west-2#) for Lambda functions
   (e.g. `versionary-lambdas`), used to package the CloudFormation template. It must be unique across all AWS accounts.
   Also, you'll need to update the name in the Makefile `package` command.
2. [Verify your domain](https://us-west-2.console.aws.amazon.com/ses/home?region=us-west-2#verified-senders-domain:)
   for sending email via AWS Simple Email Service (SES). If you're using AWS Route 53
   to [manage your domain](https://console.aws.amazon.com/route53/home?region=us-west-2#hosted-zones:),
   this is a trivial exercise, and happens very quickly.
3. Build the applications for running in your local development environment:

```bash
make build
```

4. Create tables in DynamoDB for the Versionary API in your development environment:

```bash
./ops table --env dev
```

5. Create a bootstrap admin user account and bearer token. Make a note of the token for exploring the API:

```bash
./ops user create --env dev --admin --email username@example.com --password password --familyname Family --givenname Given
./ops token create --env dev username@example.com
```

6. Run the API locally:

```bash
./api --env dev
```

7. Explore the API with [Postman](https://www.postman.com/), or a similar tool. You'll need to set the `Authorization`
   header to `Bearer <token>`, where `<token>` is the token you created previously. For simple GET requests, you can use
   the [ModHeader)](https://modheader.com/) extension for Chrome or Firefox. Also, be sure to check out
   the Swagger [API documentation](http://localhost:8080/docs).

## Release process

A CloudFormation template ("infrastructure as code") is used to create and manage most of the following infrastructure
elements. Exceptions include DynamoDB tables (created using an operations command), SSL Certificates, Route 53 DNS
records, etc.

* GitHub code repository
* CloudFormation template
* API Gateway HTTP Proxy
* Versionary API Lambda Function
* DynamoDB Tables (created using an operations command)

You can use different git branches (e.g. qa, staging, prod) to manage the code running in different operating
environments. Once you have code ready for release to a particular environment, you can use the following steps
to deploy the code to that environment:

1. Ensure that you have the desired git branch checked out locally, and that all tests pass.
2. Run `make build` to build the `./api` and `./ops` commands for local use.
3. Run `./ops table --env <env>` to create any missing DynamoDB tables in the environment.
4. Run `make deploy env=[qa|staging|prod]` to build release artifacts and deploy the CloudFormation template.
5. Test the updated code running in the specified environment.
