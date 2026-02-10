# Stage 1: Build Frontend
FROM node:20-bookworm AS frontend-builder

WORKDIR /app/website
COPY website/package*.json ./
RUN npm install
COPY website/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.23-bookworm AS backend-builder

WORKDIR /app/server
COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
COPY --from=frontend-builder /app/website/build ./internal/api/web/dist/

RUN CGO_ENABLED=1 go build -o lectures ./cmd/server

# Stage 3: Runtime
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    ghostscript \
    poppler-utils \
    libreoffice-writer \
    libreoffice-impress \
    pandoc \
    curl \
    ca-certificates \
    fontconfig \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN curl --proto '=https' --tlsv1.2 -fsSL https://drop-sh.fullyjustified.net | sh \
    && mv tectonic /usr/local/bin/

RUN mkdir -p /data/files /data/models
VOLUME /data

COPY --from=backend-builder /app/server/lectures /usr/local/bin/lectures
COPY server/prompts /prompts
COPY server/xelatex-template.tex /xelatex-template.tex

ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=3000

EXPOSE 3000

ENTRYPOINT ["lectures", "-configuration", "/data/configuration.yaml"]
