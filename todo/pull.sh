#!/usr/bin/env bash

git submodule update --init --remote --recursive

cd ./src/proto/protocol
git pull origin main
cd -

cd ./tools/auto-export-excel-tool
git pull origin main
cd -

git pull origin main