FROM golang:1.22.5-alpine3.20

WORKDIR /go/src/app

# Copy files
# See Also: .dockerignore
COPY . .

RUN go mod tidy && \
    go build -o ./artifact-server -ldflags="-s -w" ./cmd/server/...;

EXPOSE 8080
CMD ["./artifact-server"]
