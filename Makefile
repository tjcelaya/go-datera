VERSION ?= v0.1.0
NAME=dat-csi-plugin

compile:
	@echo "==> Building the Datera Golang SDK"
	@env go get -d ./...
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${NAME}
	@env go vet ./...

compile-local:
	@echo "==> Building the Datera Golang SDK locally"
	@env CGO_ENABLED=0 GOARCH=amd64 go build -o ${NAME}
	@env go vet ./...

clean:
	@echo "==> Cleaning artifacts"
	@GOOS=linux go clean -i -x ./...
