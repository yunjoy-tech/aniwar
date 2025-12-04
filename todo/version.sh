#!/usr/bin/env bash

go build -gcflags "-N -l" %RACE% -o ./output/bin/linux/versionTool ./tools/version/
