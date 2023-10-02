#!/usr/bin/env bash

set -eo pipefail

PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE}")"/..; pwd)

GENERATE_PATHS="./internal/server/*.proto"

for p in $(ls ${GENERATE_PATHS}); do
	this_dir=$(dirname $p)
	protoc -I=${this_dir} -I=${PROJECT_ROOT}/external/proto -I=${PROJECT_ROOT}/dist/protoc-include --go_out=${this_dir} --go_opt=paths=source_relative $p
done
