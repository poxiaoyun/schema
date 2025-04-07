
BUILD_DATE?=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_VERSION?=$(shell git describe --tags --dirty --abbrev=0 2>/dev/null || git symbolic-ref --short HEAD)
GIT_COMMIT?=$(shell git rev-parse HEAD 2>/dev/null)
GIT_BRANCH?=$(shell git symbolic-ref --short HEAD 2>/dev/null)
VERSION?=$(shell echo "${GIT_VERSION}" | sed -e 's/^v//')

BIN_DIR?=bin
IMAGE_REGISTRY?=registry.cn-hangzhou.aliyuncs.com
IMAGE_REPOSITORY?=xiaoshiai
IMAGE_NAME?=$(shell basename $(CURDIR))

LDFLAGS+=-w -s
LDFLAGS+=-X 'xiaoshiai.cn/common/version.gitVersion=${GIT_VERSION}'
LDFLAGS+=-X 'xiaoshiai.cn/common/version.gitCommit=${GIT_COMMIT}'
LDFLAGS+=-X 'xiaoshiai.cn/common/version.buildDate=${BUILD_DATE}'
BUILD_TARGET?=./cmd/...
##@ All
all: build ## build all

define build-binary
	@echo "Building ${1}-${2}";
	@mkdir -p ${BIN_DIR}/${1}-${2};
	GOOS=${1} GOARCH=$(2) CGO_ENABLED=0 go build -gcflags=all="-N -l" -ldflags="${LDFLAGS}" -o ${BIN_DIR}/${1}-${2} $(BUILD_TARGET)
endef
##@ Build
.PHONY: build
build: ## Build local binary.
	$(call build-binary,linux,amd64)
	$(call build-binary,linux,arm64)

clean:
	rm -rf ${BIN_DIR}
 