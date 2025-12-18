# Makefile for embedding build info into the executable

BUILD_DATE = $(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DISTRO ?= $(shell lsb_release -sd)

USE_UPX ?= 0
ifneq ($(USE_UPX),0)
	BUILD_PARTS := build compress
else
	BUILD_PARTS := build
endif

GOALS := amd64 arm5 arm6 arm7 arm64 clean
ifeq (,$(filter $(GOALS),$(MAKECMDGOALS)))
  $(error Build with make amd64/arm5/arm6/arm7/arm64)
endif

amd64:
amd64: export GOOS=linux
amd64: export GOARCH=amd64
amd64: export FLAVOUR=$(GOOS)-$(GOARCH)
amd64: $(BUILD_PARTS)

arm5: export GOOS=linux
arm5: export GOARCH=arm
arm5: export GOARM=5
arm5: export FLAVOUR=$(GOOS)-$(GOARCH)$(GOARM)
arm5: $(BUILD_PARTS)

arm6: export GOOS=linux
arm6: export GOARCH=arm
arm6: export GOARM=6
arm6: export FLAVOUR=$(GOOS)-$(GOARCH)$(GOARM)
arm6: $(BUILD_PARTS)

arm7: export GOOS=linux
arm7: export GOARCH=arm
arm7: export GOARM=7
arm7: export FLAVOUR=$(GOOS)-$(GOARCH)$(GOARM)
arm7: $(BUILD_PARTS)

arm64: export GOOS=linux
arm64: export GOARCH=arm64
arm64: export FLAVOUR=$(GOOS)-$(GOARCH)
arm64: $(BUILD_PARTS)

builddir: $(FLAVOUR)
	mkdir -p build/$(FLAVOUR)

# -s disable symbol table
# -w disable DWARF generation
build: builddir
	go build -o build/$(FLAVOUR)/sforwarder -ldflags "-w -s -X 'main.ApplicationBuildDate=$(BUILD_DATE)' -X 'main.ApplicationBuildDistro=$(BUILD_DISTRO)'"

# upx will make the binary much smaller, but will take a while
compress: build
	upx --brute build/$(FLAVOUR)/sforwarder

clean:
	rm -Rf build

.PHONY: clean
