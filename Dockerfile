ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm AS builder

ARG MML_CLUSTER_OBSERVABILITY_ENABLED=false
ENV MML_CLUSTER_OBSERVABILITY_ENABLED=$MML_CLUSTER_OBSERVABILITY_ENABLED

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go run ./cmd/mermaidlive -transpile && CGO_ENABLED=0 go build --tags=embed -v -o /run-app ./cmd/mermaidlive

FROM alpine:latest AS alpine
# create a user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
# prepare data dir
RUN mkdir /appdata && chown appuser:appgroup /appdata

FROM scratch
WORKDIR /


# copy the user
COPY --from=alpine /etc/passwd /etc/passwd
COPY --chown=appuser:appgroup --from=alpine /appdata /appdata
USER appuser

COPY --from=builder /run-app /
ENTRYPOINT [ "./run-app" ]
