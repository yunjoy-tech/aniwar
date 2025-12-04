#!/usr/bin/env bash
Arg=$1
tail -f log/plog/*${Arg}*
