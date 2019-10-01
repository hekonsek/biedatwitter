all: format silent-test build

build:
	GO111MODULE=on go build biedatwitter.go

test: build
	GO111MODULE=on go test -v

silent-test:
	GO111MODULE=on go test

format:
	GO111MODULE=on go fmt *.go

docker-build: build
	docker build -t hekonsek/biedatwitter .

docker-push: docker-build
	docker push hekonsek/biedatwitter