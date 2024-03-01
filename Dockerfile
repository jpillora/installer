# syntax=docker/dockerfile:1.4
FROM golang:1.22.0-alpine3.18 AS base
USER 1001
ENV GOPATH=/tmp/go
ENV GOCACHE=/tmp/go-cache
WORKDIR /tmp/app
COPY . .
RUN go mod download -x

RUN CGO_ENABLED=0 go build -o /tmp/bin/bin-installer ./main.go
RUN chmod +x /tmp/bin/bin-installer

FROM gcr.io/distroless/static-debian11:nonroot
LABEL org.opencontainers.image.source=https://github.com/kloudlite/bin-installer
COPY --from=base /tmp/bin/bin-installer ./bin-installer
CMD ["./bin-installer"]
