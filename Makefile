all: build

MODULE = "github.com/LukeEuler/funnel-log-reporter"

GOIMPORTS := $(shell command -v goimports 2> /dev/null)
CILINT := $(shell command -v golangci-lint 2> /dev/null)

style:
ifndef GOIMPORTS
	$(error "goimports is not available please install goimports")
endif
	! find . -path ./vendor -prune -o -name '*.go' -print | xargs goimports -d -local ${MODULE} | grep '^'

format:
ifndef GOIMPORTS
	$(error "goimports is not available please install goimports")
endif
	find . -path ./vendor -prune -o -name '*.go' -print | xargs goimports -l -local ${MODULE} | xargs goimports -l -local ${MODULE} -w

cilint:
ifndef CILINT
	$(error "golangci-lint is not available please install golangci-lint")
endif
	golangci-lint run --timeout 5m0s

test: style
	go test -cover ./...

build: test
	go build -o build/flr app/main.go

linux: test
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/flr app/main.go

.PHONY: linux style format cilint test build
