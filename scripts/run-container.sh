#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

docker run --rm -p "8080:8080" mermaidlive
