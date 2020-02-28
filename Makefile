.PHONY: cli
cli:
	rm -f tutil
	go build -o ./tutil ./cmd/tutil


.PHONY: testenv
testenv:
	( cd testenv ; make testenv )

.PHONY: stop-testenv
stop-testenv:
	( cd testenv ; make clean )

.PHONY: test
test: vendor
	go test -race -cover ./...

.PHONY: lint
lint: vendor
	golint $(go list ./... | grep -v /vendor/)


# cleanup dependencies and download missing ones
.PHONY: deps
deps:
	go mod tidy
	go mod download

# run dependency cleanup, followed by updating the patch version
.PHONY: deps-update
deps-update: deps
	go get -u=patch
	
# run tests
.PHONY: tests
tests:
	go test -race -cover -count 1 ./...

# run standard go tooling for better rcode hygiene
.PHONY: tidy
tidy: imports fmt
	go vet ./...
	golint ./...

# automatically add missing imports
.PHONY: imports
imports:
	find . -type f -name '*.go' -exec goimports -w {} \;

# format code and simplify if possible
.PHONY: fmt
fmt:
	find . -type f -name '*.go' -exec gofmt -s -w {} \;

verifiers: staticcheck

staticcheck:
	@echo "Running $@ check"
	@GO111MODULE=on ${GOPATH}/bin/staticcheck ./...
