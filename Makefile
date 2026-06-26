.PHONY: build lint clean

lint:
	golangci-lint run

build:
	mkdir -p build
	GOBIN=$(PWD)/build go install ./cmd/...

clean:
	rm -rf build
