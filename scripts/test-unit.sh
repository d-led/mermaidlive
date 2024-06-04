#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

go test -v ./...
