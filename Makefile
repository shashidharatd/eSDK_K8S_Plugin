
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
SUFFIX:=$(GOOS)-$(GOARCH)


all: build

build:
	mkdir -p bin
	go build -o bin/passwd-encrypt-$(SUFFIX) ./cmd/passwd-encrypt
	go build -o bin/huawei-csi-$(SUFFIX) ./cmd/huawei-csi

.phony: all
