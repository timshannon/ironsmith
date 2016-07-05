#!/bin/bash

go-bindata web/... && go build -a -o ironsmith
