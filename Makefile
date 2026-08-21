.PHONY: build test vet clean run

build:
	go build -o robotdemo ./cmd/robotdemo

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

clean:
	rm -f robotdemo

run:
	go run ./cmd/robotdemo
