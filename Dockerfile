# Build the manager binary
FROM golang:1.18 as builder

WORKDIR vcluster
COPY . .

RUN CGO_ENABLED=0 go build -mod vendor -o /plugin main.go


FROM alpine

WORKDIR /
COPY --from=builder /plugin .

ENTRYPOINT ["/plugin"]
