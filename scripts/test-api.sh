#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

go test -tags=api_test -v  ./...
