build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o s3-object-cache

container: build
	docker build -t fwieffering/s3-object-cache .

sidecar-container:
	make -C sidecar container
