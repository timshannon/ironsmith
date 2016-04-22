#!/bin/bash

echo $PATH

go-bindata web/... && go build -a -v -o ironsmith
