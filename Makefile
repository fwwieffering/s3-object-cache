build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o s3-object-cache

container: build
	docker build -t fwwieffering/s3-object-cache .

sidecar:
	make -C sidecar container
