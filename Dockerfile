ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go run . -transpile && CGO_ENABLED=0 go build --tags=embed -v -o /run-app .

FROM alpine:latest as alpine
# create a user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

FROM scratch
WORKDIR /

# copy the user
COPY --from=alpine /etc/passwd /etc/passwd
USER appuser

COPY --from=builder /run-app /
CMD ["./run-app"]
