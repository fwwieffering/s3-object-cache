FROM alpine

RUN apk --no-cache add ca-certificates

WORKDIR /app/


COPY ./map-service-sidecar .
CMD ["./map-service-sidecar"]
