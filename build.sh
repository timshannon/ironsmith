#!/bin/bash
go-bindata web/... && go build -a -v -o ironsmith
