#!/bin/bash

# Run this script to build a genblog release.

set -euo pipefail

# Generated binaries and README.md are ignored by Git.
GOOS=linux GOARCH=amd64 go build -o release/bin/linux .
GOOS=darwin GOARCH=amd64 go build -o release/bin/mac .
cp README.md release

line_number() {
  grep -n "$1" release/README.md | cut -f1 -d:
}

from=$(line_number "# genblog")
to=$(line_number '## Write')
to=$((to - 1))

sed -i '' "$from","$to"d release/README.md

prepend() {
  # shellcheck disable=SC2059
  (printf "$1"; cat "$2") > tmp
  mv tmp "$2"
}

prepend "# Blog\n\nA static blog.\n\n" release/README.md

tar -czf blog.tar.gz release
