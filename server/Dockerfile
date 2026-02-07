# Stage 1: Build the Go application
FROM golang:1.23-bookworm AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o lectures ./cmd/server

# Stage 2: Runtime image with all dependencies
FROM debian:bookworm-slim

# Avoid prompts during installation
ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
# 1. ffmpeg: Audio extraction
# 2. ghostscript: PDF page extraction
# 3. libreoffice: PPTX/DOCX conversion
# 4. pandoc: Markdown conversion
# 5. curl/ca-certificates: For downloading models
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ghostscript \
    libreoffice-writer \
    libreoffice-impress \
    pandoc \
    curl \
    ca-certificates \
    fontconfig \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Tectonic (LaTeX engine)
RUN curl --proto '=https' --tlsv1.2 -fsSL https://drop-sh.fullyjustified.net | sh \
    && mv tectonic /usr/local/bin/

# Create data directory
RUN mkdir -p /data/files /data/models
VOLUME /data

# Copy the binary from builder
COPY --from=builder /app/lectures /usr/local/bin/lectures

# Set environment variables
ENV STORAGE_DATA_DIRECTORY=/data
ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=3000

EXPOSE 3000

# Run the server
# We use the -configuration flag if needed, but the server creates one by default in ~/.lectures
# We'll point it to /data/configuration.yaml
ENTRYPOINT ["lectures", "-configuration", "/data/configuration.yaml"]