# ---------- Builder ----------
FROM golang:1.26.1 AS builder
WORKDIR /src

# 1️⃣ Install build‑time packages (only if you need git, gcc etc.)
# RUN apk add --no-cache git

# 2️⃣ Cache go.mod / go.sum
COPY go.mod go.sum ./
RUN go mod download

# 3️⃣ Copy only source code you actually need
COPY . .
# If you have additional packages, add them here

# 4️⃣ Build static binary
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH}
RUN go build -ldflags="-s -w" -o /out/server ./cmd/server

# ---------- Runtime ----------
FROM alpine:latest AS runtime
WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /out/server /app/server

# Declare the ports your app will listen on
EXPOSE 8080 50051

# Run the binary
ENTRYPOINT ["/app/server"]
