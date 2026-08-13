# syntax=docker/dockerfile:1

FROM golang:1.25.7 AS build

WORKDIR /src

# Cache dependencies independently from application source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go test ./...

ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/server /server
COPY --from=build /out/migrate /migrate

EXPOSE 8080

ENTRYPOINT ["/server"]
