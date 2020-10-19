#/bin/bash

if [ -z "$1" ]
then
    echo "no version: https://eu-central-1.console.aws.amazon.com/ecr/repositories/imaginary/?region=eu-central-1"
    exit 0
fi

aws ecr get-login-password --region eu-central-1 | docker login --username AWS --password-stdin 421016705922.dkr.ecr.eu-central-1.amazonaws.com
docker build -t 421016705922.dkr.ecr.eu-central-1.amazonaws.com/imaginary:v$1 .
docker push 421016705922.dkr.ecr.eu-central-1.amazonaws.com/imaginary:v$1


az acr login -n imaginary
docker tag 421016705922.dkr.ecr.eu-central-1.amazonaws.com/imaginary:v$1 imaginary.azurecr.io/imaginary:v$1
docker push imaginary.azurecr.io/imaginary:v$1


echo "please update on awsus/mda, euestudy/imaginary euestudy/basel/imaginary"