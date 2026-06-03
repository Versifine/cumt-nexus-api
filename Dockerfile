# syntax=docker/dockerfile:1

ARG GO_VERSION=1.25.4

FROM golang:${GO_VERSION}-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && \
    adduser -S -G app app

COPY --from=build /out/api /app/api
COPY --from=build /out/migrate /app/migrate
COPY migrations /app/migrations

RUN mkdir -p /app/var/uploads && \
    chown -R app:app /app

USER app

EXPOSE 8080

ENTRYPOINT ["/app/api"]
