#!/bin/bash

dir=$1

export GOPATH=$dir 

echo working dir: $dir

go get -u git.townsourced.com/townsourced/ironsmith

go-bindata web/... && go build -a -v -o ironsmith
