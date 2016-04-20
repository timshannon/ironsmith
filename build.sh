#!/bin/bash

dir=$1

go get -u git.townsourced.com/townsourced/ironsmith

ls $dir
#cd $dir/src/git.townsourced.com/townsourced/ironsmith

go-bindata web/... && go build -a -v -o ironsmith
