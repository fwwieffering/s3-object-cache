build: deps
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o s3-object-cache-sidecar

deps:
	dep ensure

container: build
	docker build -t fwieffering/s3-object-cache-sidecar .

run: container
	docker run -p 8080:80 fwwieffering/s3-object-cache-sidecar
