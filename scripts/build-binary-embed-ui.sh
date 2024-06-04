#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

echo "--== transpiling ==--"
go run ./cmd/mermaidlive -transpile

echo "--== building mermaidlive binary ==--"
go build --tags=embed ./cmd/mermaidlive

