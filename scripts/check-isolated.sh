#!/bin/sh
# Check the candidate modules without go.work or a previously published tag.
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
check_dir=$(mktemp -d "${TMPDIR:-/tmp}/shape-isolated.XXXXXX")
trap 'rm -rf "$check_dir"' EXIT HUP INT TERM

cd "$repo_dir"
GOWORK=off go test ./...
GOWORK=off go vet ./...

cp tools/shapevet/go.mod "$check_dir/go.mod"
cp tools/shapevet/go.sum "$check_dir/go.sum"
cd tools/shapevet
# The only replacement is the candidate root module; external dependencies
# still resolve through the tool's committed module and checksum files.
GOWORK=off go mod edit -modfile="$check_dir/go.mod" -replace="github.com/rhevorn/shape=$repo_dir"
GOWORK=off go mod download -modfile="$check_dir/go.mod"
GOWORK=off go mod verify -modfile="$check_dir/go.mod"
GOWORK=off go test -mod=readonly -modfile="$check_dir/go.mod" ./...
GOWORK=off go vet -mod=readonly -modfile="$check_dir/go.mod" ./...
GOWORK=off go build -mod=readonly -modfile="$check_dir/go.mod" -o "$check_dir/shapevet" .
