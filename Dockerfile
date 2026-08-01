FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN go generate ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.Version=${VERSION}" -o /helmfmt .

FROM scratch
COPY --from=build /helmfmt /usr/local/bin/helmfmt

WORKDIR /work
ENTRYPOINT ["/usr/local/bin/helmfmt"]
