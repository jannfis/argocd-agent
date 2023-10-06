#!/bin/sh

set -eo pipefail

go_packages=$(go list ./...)
for p in $go_packages; do
	mkdir -p test/profile/$pkg
done
