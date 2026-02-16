# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/website
COPY website/package*.json ./
RUN npm install
COPY website/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.24-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app/server
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o lectures-assistant ./cmd/server

# Stage 3: Runtime
FROM debian:bookworm-slim

# Avoid prompts during installation
ENV DEBIAN_FRONTEND=noninteractive

# Install system dependencies
# 1. ffmpeg: Audio extraction
# 2. ghostscript: PDF page extraction
# 3. libreoffice-writer-nogui: Headless PPTX/DOCX conversion (much smaller)
# 4. curl/ca-certificates: For downloading models and tectonic
# 5. fontconfig: For font management
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ghostscript \
    libreoffice-writer-nogui \
    libreoffice-impress-nogui \
    curl \
    ca-certificates \
    fontconfig \
    tar \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Tectonic (LaTeX engine) - Fixed architecture-aware installation
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then TECTONIC_ARCH="x86_64-unknown-linux-gnu"; \
    elif [ "$ARCH" = "aarch64" ]; then TECTONIC_ARCH="aarch64-unknown-linux-musl"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -L "https://github.com/tectonic-typesetting/tectonic/releases/download/tectonic%400.15.0/tectonic-0.15.0-${TECTONIC_ARCH}.tar.gz" | tar -xz -C /usr/local/bin/

# Install Pandoc (Static Binary) - Much smaller than apt version
RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then PANDOC_ARCH="amd64"; \
    elif [ "$ARCH" = "aarch64" ]; then PANDOC_ARCH="arm64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -L "https://github.com/jgm/pandoc/releases/download/3.1.11/pandoc-3.1.11-linux-${PANDOC_ARCH}.tar.gz" | tar -xz --strip-components=1 -C /usr/local/

# Create application directories
RUN mkdir -p /data/files /data/models /app/www /app/prompts
VOLUME /data

# Copy the binary from builder
COPY --from=backend-builder /app/server/lectures-assistant /usr/local/bin/lectures-assistant

# Copy built frontend assets
COPY --from=frontend-builder /app/website/build /app/www

# Copy resources
COPY server/prompts /app/prompts
COPY server/xelatex-template.tex /app/xelatex-template.tex

# Set environment variables
ENV STORAGE_DATA_DIRECTORY=/data
ENV STORAGE_WEB_DIRECTORY=/app/www
ENV IN_DOCKER_ENV=true
ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=3000

# Ensure the server can find prompts and template in their new locations
WORKDIR /app

EXPOSE 3000

# Run the server
ENTRYPOINT ["lectures-assistant", "-configuration", "/data/configuration.yaml"]
