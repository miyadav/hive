#!/bin/bash
set -e

REPO_URL="https://github.com/openshift/hive.git"
CLONE_DIR="hive-callvis-tmp"
OUTPUT_DIR="callvisdocs/callvis"
TARGET_PKG="github.com/openshift/hive/cmd/hiveadmission"
IGNORE_PKGS="github.com/openshift/hive/cmd/manager,github.com/openshift/hive/cmd/operator,github.com/openshift/hive/contrib/cmd/hiveutil,github.com/openshift/hive/contrib/cmd/waitforjob,github.com/openshift/hive/hack"

DOT_FILE="$OUTPUT_DIR/callgraph.dot"
SVG_FILE="$OUTPUT_DIR/callgraph.svg"

echo "==> Cloning repository to clean workspace..."
rm -rf "$CLONE_DIR"
git clone "$REPO_URL" "$CLONE_DIR"

cd "$CLONE_DIR"

echo "==> Removing problematic imports (fsnotify, golang.org/x/sys/unix)..."
# Optional: smarter grep+sed filtering to comment out problematic imports
find . -type f -name "*.go" -exec sed -i 's|"github.com/fsnotify/fsnotify"|"github.com/fsnotify/fsnotify_disabled"|' {} +
find . -type f -name "*.go" -exec sed -i 's|"golang.org/x/sys/unix"|"golang.org/x/sys/unix_disabled"|' {} +

echo "==> Tidying modules..."
go mod tidy

echo "==> Preparing output directory..."
mkdir -p "../$OUTPUT_DIR"

echo "==> Generating call graph..."
GOFLAGS=-mod=mod go-callvis -tests -format dot -group pkg,type -focus "$TARGET_PKG" -ignore "$IGNORE_PKGS" "$TARGET_PKG" > "../$DOT_FILE"

dot -Tsvg "../$DOT_FILE" -o "../$SVG_FILE"

cd ..
rm -rf "$CLONE_DIR"

echo "✅ Call graph generated:"
echo "DOT:  $DOT_FILE"
echo "SVG:  $SVG_FILE"

