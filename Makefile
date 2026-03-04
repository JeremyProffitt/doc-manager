.PHONY: all build build-DocManagerFunction test test-integration vet sam-build sam-deploy seed local clean

all: vet test build

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap cmd/lambda/main.go

build-DocManagerFunction:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o $(ARTIFACTS_DIR)/bootstrap cmd/lambda/main.go
	cp -r templates $(ARTIFACTS_DIR)/templates
	cp -r static $(ARTIFACTS_DIR)/static

test:
	go test -race ./...

test-integration:
	go test -tags=integration -race ./...

vet:
	go vet ./...

sam-build:
	sam build

sam-deploy:
	sam deploy --no-confirm-changeset --no-fail-on-empty-changeset

seed:
	go run cmd/seed/main.go

local:
	go run cmd/lambda/main.go

clean:
	rm -f bootstrap
