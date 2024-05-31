#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

mkdir -p "$(pwd)/functions"
go run . -transpile
GOBIN=$(pwd)/functions go install --tags=embed -v .
