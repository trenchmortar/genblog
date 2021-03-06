#!/bin/bash

# Run this script to build a genblog release.

set -euo pipefail

# Generated binaries are ignored by Git.
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o blog/bin/linux .
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o blog/bin/mac .

# Compress
(
  cd blog/bin

  if ! command -v upx >/dev/null; then
    if ! command -v brew >/dev/null; then
      echo "install upx and then re-run script"
      exit 1
    fi

    brew install upx
  fi

  upx linux
  upx mac
)

# Generated README.md is ignored by Git.
cp README.md blog

line_number() {
  grep -n "$1" blog/README.md | cut -f1 -d:
}

from=$(line_number "# genblog")
to=$(line_number '## Write')
to=$((to - 1))

sed -i '' "$from","$to"d blog/README.md

prepend() {
  # shellcheck disable=SC2059
  (printf "$1"; cat "$2") > tmp
  mv tmp "$2"
}

prepend "# Blog\n\nA static blog.\n\n" blog/README.md

tar -czf blog.tar.gz blog
