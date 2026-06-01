# SFS (SmallFileSync) — TUI file sync tool
# NOTE: SFS is a TUI tool. Run with `-it` flags for TTY:
#   docker run -it sfs

# ---- Build stage ----
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /sfs .

# ---- Runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /sfs /usr/local/bin/sfs

# SFS stores config in /root/.config/small-filesync (Linux default)
VOLUME ["/root/.config/small-filesync"]

ENTRYPOINT ["sfs"]
