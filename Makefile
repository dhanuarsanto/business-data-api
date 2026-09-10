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

build:
	go build -o $(BIN_NAME) ./cmd/api

test:
	go test ./... -v

clean:
	go clean
	-$(CLEAN_CMD)
