BINARY := nbt-cli
CMD := ./cmd/nbt-cli
BIN_DIR := $(CURDIR)/bin

.PHONY: build install clean

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) $(CMD)

install:
	go install $(CMD)

clean:
	rm -rf $(BIN_DIR)


