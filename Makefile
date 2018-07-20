BIN := ethereum-tps

.PHONY: all
all: build

.PHONY: build
build:
	go build -i -o ${BIN} ./cmd/ethereum-tps