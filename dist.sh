#!/bin/bash

set -eu

cd $(dirname $0)

rm -rf dist
mkdir -p dist

build() {
  export GOOS=$1
  export GOARCH=$2
  export CGO_ENABLED=0
  FILENAME="imapdownloader-${GOOS}-${GOARCH}"
  go build -a -installsuffix cgo  -o "dist/${FILENAME}" .
}

build linux amd64
build windows amd64
build darwin amd64

build linux arm64
build linux loong64
build windows arm64
build darwin arm64

