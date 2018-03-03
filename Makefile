REGISTRY=acoshift/redirect-backend
TAG=1.0

dev:
	go run main.go

clean:
	rm -f redirect-backend

build:
	env GOOS=linux GOARCH=amd64 go build -o redirect-backend -ldflags "-w -s" main.go

docker: clean build docker-build docker-push

docker-build:
	docker build -t $(REGISTRY):$(TAG) .

docker-push:
	docker push $(REGISTRY):$(TAG)
