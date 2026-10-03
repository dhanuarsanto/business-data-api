ifeq ($(OS),Windows_NT)
	BIN_NAME = bin/api.exe
	CLEAN_CMD = if exist bin rmdir /s /q bin & if exist tmp rmdir /s /q tmp & del /q *.exe 2>nul
else
	BIN_NAME = bin/api
	CLEAN_CMD = rm -rf bin tmp *.exe
endif

run:
	go run ./cmd/api

dev:
	air

VERSION := $(shell type version.txt)

build:
ifeq ($(OS),Windows_NT)
	set CGO_ENABLED=0 && go build -trimpath -ldflags="-s -w -X 'go.internal/business-data-api/internal/version.Version=$(VERSION)'" -o $(BIN_NAME) ./cmd/api
else
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X 'go.internal/business-data-api/internal/version.Version=$(VERSION)'" -o $(BIN_NAME) ./cmd/api
endif

test:
	go test ./... -v

clean:
	go clean
	-$(CLEAN_CMD)
