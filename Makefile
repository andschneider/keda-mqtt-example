ARCH       ?=amd64
CGO        ?=0
TARGET_OS  ?=linux
GO_BUILD_VARS= GO111MODULE=on CGO_ENABLED=$(CGO) GOOS=$(TARGET_OS) GOARCH=$(ARCH)

.PHONY: build
build: main.go
	${GO_BUILD_VARS} go build -o bin/keda-mqtt main.go

.PHONY: clean
clean:
	rm bin/keda-mqtt

