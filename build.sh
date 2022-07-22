#!/bin/bash

cd "$(dirname "$0")"
export GOPATH=$PWD

set -ex
rm -rf tmp
mkdir -p tmp
GOOS=linux GOARCH=386 go build -o tmp/webdir-linux32.bin main.go
GOOS=windows GOARCH=386 go build -o tmp/webdir-win32.exe main.go
GOOS=darwin GOARCH=amd64 go build -o tmp/webdir-mac64.bin main.go

chmod +x tmp/*
upx -9 tmp/*

