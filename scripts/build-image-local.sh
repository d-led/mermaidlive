#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

docker build -t mermaidlive --build-arg GO_VERSION=1.22 .
