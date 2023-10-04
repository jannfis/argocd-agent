#!/usr/bin/env bash

set -eo pipefail

PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE}")"/..; pwd)

GENERATE_PATHS="
	${PROJECT_ROOT}/server/auth/*.proto
"

for p in ${GENERATE_PATHS}; do
	set -x
	this_dir=$(dirname $p)
	api_name=$(basename $this_dir)
	this_package=$(basename $p)
	echo "--> Generating Protobuf and gRPC client for $this_package"
	protoc  -I=${this_dir} \
		-I=${PROJECT_ROOT}/external/proto \
		-I=${PROJECT_ROOT}/dist/protoc-include \
	       	--go_out=${PROJECT_ROOT}/pkg/api/grpc/${api_name} \
	       	--go_opt=paths=source_relative \
		--go-grpc_out=${PROJECT_ROOT}/pkg/api/grpc/${api_name} \
		--go-grpc_opt=paths=source_relative \
		$p
done
