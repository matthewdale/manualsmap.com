# Build image
FROM golang:1.14-alpine as builder
WORKDIR /go/src/github.com/matthewdale/manualsmap.com/

COPY cmd cmd/
COPY encoders encoders/
COPY handlers handlers/
COPY go.mod .
COPY go.sum .

RUN GOOS=linux go build -o api ./cmd/api.go


# Run image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /go/src/github.com/matthewdale/manualsmap.com/api .
COPY public public/
COPY api.sh .
CMD ["sh", "api.sh"]
