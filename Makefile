.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	mkdir -p test/out
	./hack/test.sh

.PHONY: codegen
codegen: protogen

.PHONY: protogen
protogen:
	./hack/generate-proto.sh
