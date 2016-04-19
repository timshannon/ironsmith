#!/bin/bash
mkdir -p ./workspace/src
export GOPATH=./workspace

git clone https://git.townsourced.com/tshannon/config.git ./workspace/src

go-bindata web/... && go build -a -v -o ironsmith -pkgdir ./workspace
