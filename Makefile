BINARY_NAME := octocatalog
GO := go

.PHONY: build test lint ci clean

build:
	$(GO) build -o $(BINARY_NAME) .

test:
	$(GO) test -v -cover ./...

lint:
	$(GO) vet ./...

ci: build test lint

clean:
	rm -f $(BINARY_NAME)
