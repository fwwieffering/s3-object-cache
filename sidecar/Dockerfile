FROM alpine

RUN apk --no-cache add ca-certificates

WORKDIR /app/


COPY ./s3-object-cache-sidecar .
CMD ["./s3-object-cache-sidecar"]
