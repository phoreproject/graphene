#!/bin/bash

if [ ! -z "$TRAVIS_TAG" ] &&
    [ "$TRAVIS_PULL_REQUEST" == "false" ]; then
  echo "This will deploy!"

  # Cross-compile for all platforms
  export CGO_ENABLED=1
  docker pull karalabe/xgo-latest
  go get github.com/karalabe/xgo
  mkdir dist/ && cd dist/
  xgo --targets=windows/amd64,darwin/amd64,linux/amd64 ../cmd/beacon
  xgo --targets=windows/amd64,darwin/amd64,linux/amd64 ../cmd/validator
  xgo --targets=windows/amd64,darwin/amd64,linux/amd64 ../cmd/keygen
  chmod +x *
  # Upload to GitHub Release page
  ghr --username phoreproject -t $GITHUB_TOKEN --replace --prerelease --debug $TRAVIS_TAG .
else
  echo "This will not deploy!"
fi
