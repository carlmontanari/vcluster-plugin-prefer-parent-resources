FROM golang:1.19 as builder

WORKDIR /plugin

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY main.go main.go
COPY prefer-parent-resources/ prefer-parent-resources/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o plugin main.go

FROM gcr.io/distroless/static-debian11:nonroot

ENV PLUGIN_NAME="prefer-parent-resources-hooks"

WORKDIR /
COPY --from=builder /plugin/plugin .

ENTRYPOINT ["/plugin"]
