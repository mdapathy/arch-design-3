#!/usr/bin/env sh

cd ./integration || return
CGO_ENABLED=0 boodtdma
cat ./out/reports/test.txt
