all: build

.PHONY: vendor
vendor:
	dep ensure

.PHONY: build
build: vendor
	go build ./...

.PHONY: testenv
testenv:
	( cd testenv ; make testenv )

.PHONY: stop-testenv
stop-testenv:
	( cd testenv ; make stop-testenv )

.PHONY: test
test: vendor
	go test -race -cover ./...

.PHONY: lint
lint: vendor
	golint $(go list ./... | grep -v /vendor/)
