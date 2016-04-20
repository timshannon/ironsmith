#!/bin/bash

go get -u git.townsourced.com/townsourced/ironsmith

go-bindata web/... && go build -a -v -o ironsmith
