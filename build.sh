#!/bin/bash
git clone https://git.townsourced.com/tshannon/config.git ./workspace/src

go-bindata web/... && go build -a -v -o ironsmith -pkgdir ./workspace
