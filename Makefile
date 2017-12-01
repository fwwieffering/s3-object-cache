
run:
	go install
	s3-object-cache

redis:
	docker run --rm -d --name s3-cache-redis -p 6379:6379 redis

clean:
	rm -rf xtencils/
	docker stop s3-cache-redis
