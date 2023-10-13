.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	mkdir -p test/out
	go test -race -coverprofile test/out/test.coverage ./...

.PHONY: codegen
codegen: protogen

.PHONY: protogen
protogen:
	./hack/generate-proto.sh
