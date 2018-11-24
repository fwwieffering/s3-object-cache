build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o map-service-poc

container: build
	docker build -t map-service-poc .

sidecar:
	make -C sidecar container
