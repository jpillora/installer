
.PHONY: test
test:
	go test ./...

.PHONY: build
build: bin/installer

bin/installer: main.go pkg/api/releases.go pkg/handlers/scripts/install.sh pkg/handlers/healthz.go pkg/handlers/scripts.go pkg/helpers/latest.go
	@mkdir -p bin
	go build -o bin/installer main.go

.PHONY: serve
serve: build
	./bin/installer