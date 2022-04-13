#!/usr/bin/env bash

# Validate Operating Environment
if [[ $# -eq 1 ]]; then
  ENV=$1
else
  echo "Specify which environment to deploy the application stack to:"
  echo "  deploy.sh <dev | qa | staging | prod>"
  exit 1
fi

echo "Validating template.yml"
aws cloudformation validate-template \
  --region us-west-2 \
  --template-body file://template.yml
if [ $? != 0 ]; then
  exit 1
fi

if [[ -e packaged-template.yml ]]; then
  echo "Deleting existing packaged-template.yml"
  rm packaged-template.yml
fi

echo "Packaging template.yml"
aws cloudformation package \
  --region us-west-2 \
  --template-file template.yml \
  --s3-bucket versionary-lambdas \
  --output-template-file packaged-template.yml
if [ $? != 0 ]; then
  exit 1
fi

echo "Deploying packaged template"
aws cloudformation deploy \
  --region us-west-2 \
  --template-file packaged-template.yml \
  --stack-name versionary-api-"$ENV" \
  --capabilities CAPABILITY_IAM \
  --parameter-overrides ENV="$ENV"

echo "Complete."
exit 0
