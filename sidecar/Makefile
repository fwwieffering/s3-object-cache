build: deps
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o map-service-sidecar

deps:
	dep ensure

container: build
	docker build -t map-service-sidecar .

run: container
	docker run -p 8080:80 map-service-sidecar
