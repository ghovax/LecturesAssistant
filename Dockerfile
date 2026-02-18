# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/website
COPY website/package*.json ./
RUN npm install
COPY website/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.24-alpine AS backend-builder

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
ENV IN_DOCKER_ENV=true
ENV STORAGE_DATA_DIRECTORY=/data
ENV STORAGE_WEB_DIRECTORY=/app/www
ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=3000
ENV XDG_DATA_DIRS=/usr/local/share:/usr/share

# LAYER 0: Heavy System Dependencies & External Binaries
# We install everything, download external tools, then purge build-only tools
# We DO NOT purge 'tar' as it is a core dependency for the system
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ghostscript \
    libreoffice-writer-nogui \
    libreoffice-impress-nogui \
    fontconfig \
    ca-certificates \
    curl \
    tar \
    && ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then TECTONIC_ARCH="x86_64-unknown-linux-gnu"; \
    elif [ "$ARCH" = "aarch64" ]; then TECTONIC_ARCH="aarch64-unknown-linux-musl"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    curl -L "https://github.com/tectonic-typesetting/tectonic/releases/download/tectonic%400.15.0/tectonic-0.15.0-${TECTONIC_ARCH}.tar.gz" | tar -xz -C /usr/local/bin/ && \
    if [ "$ARCH" = "x86_64" ]; then PANDOC_ARCH="amd64"; \
    elif [ "$ARCH" = "aarch64" ]; then PANDOC_ARCH="arm64"; \
    else echo "Unsupported architecture: $ARCH" && exit 1; fi && \
    mkdir -p /tmp/pandoc-install && \
    curl -L "https://github.com/jgm/pandoc/releases/download/3.9/pandoc-3.9-linux-${PANDOC_ARCH}.tar.gz" | tar -xz -C /tmp/pandoc-install && \
    cp /tmp/pandoc-install/pandoc-3.9/bin/pandoc /usr/local/bin/ && \
    rm -rf /tmp/pandoc-install && \
    # Download pandoc data files from repository (not included in binary release)
    # Install to root user's data directory where pandoc looks by default
    mkdir -p /root/.local/share/pandoc/data/templates && \
    curl -sL "https://github.com/jgm/pandoc/archive/refs/tags/3.9.tar.gz" | tar -xz -C /tmp --strip-components=1 && \
    cp -r /tmp/data/templates/* /root/.local/share/pandoc/data/templates/ && \
    rm -rf /tmp/data && \
    apt-get purge -y curl && \
    apt-get autoremove -y && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /var/cache/apt/archives/*

# Setup application structure
WORKDIR /app
RUN mkdir -p /data/files /data/models /app/www /app/prompts
VOLUME /data

# Copy built artifacts from previous stages
COPY --from=backend-builder /app/server/lectures-assistant /usr/local/bin/lectures-assistant
COPY --from=frontend-builder /app/website/build /app/www
COPY server/prompts /app/prompts
COPY server/xelatex-template.tex /app/xelatex-template.tex

# Verify pandoc data files are correctly installed
RUN echo "Testing pandoc data files..." && \
    ls -la /root/.local/share/pandoc/data/templates/passoptions.latex && \
    pandoc --print-default-data-file templates/passoptions.latex > /dev/null && \
    echo "SUCCESS: Pandoc data files are accessible"

# Test PDF generation with custom template (end-to-end validation)
RUN echo "Testing PDF generation with custom template..." && \
    echo '<p>Test content</p>' | pandoc -f html -t pdf --pdf-engine=tectonic --template /app/xelatex-template.tex -o /tmp/test-output.pdf 2>&1 && \
    rm /tmp/test-output.pdf && \
    echo "SUCCESS: PDF generation with custom template works"

EXPOSE 3000

# Run the server
ENTRYPOINT ["lectures-assistant", "-configuration", "/data/configuration.yaml"]
