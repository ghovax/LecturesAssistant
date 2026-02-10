# Lectures Assistant

An AI-powered application to transcribe and process lecture recordings. Upload audio, video, or PDF files and generate transcripts, notes, and exports with minimal effort.

## Quick Start

The easiest way to run this application is with Docker, which packages all dependencies (ffmpeg, PDF tools, etc.) in one container. If you prefer not to use Docker, you can build native binaries instead.

**1. Run the Application**

Mac/Linux: Open terminal and run `./start.sh`
Windows: Double-click `start.bat`
With packaged apps: Double-click `Lectures Assistant.app` (Mac) or `Lectures Assistant.bat` (Windows)

The application will automatically create a configuration file with defaults and start the server.

**2. Configure in Browser**

The app will start at http://localhost:3000. Configure your API key and preferences through the web interface.

To get an API key, visit https://openrouter.ai/ and create a free account. The key will start with `sk-or-v1-`.

## What This Does

- Upload audio or video files to get AI-generated transcripts
- Upload PDF documents to extract and process content
- Generate comprehensive lecture notes from recordings
- Export your work to PDF or Markdown format

All files are stored locally in the `data/` folder.

## Troubleshooting

**Port already in use**
Another application is using port 8080. Open `configuration.yaml` and change `port: 8080` to `port: 8081`, then update the same port in `docker-compose.yml`.

**Stop the application**
Press `Ctrl+C` in the terminal window.

## Distributable Packages

For users who want a simple double-click experience:

**Mac:**
Run `./package-mac.sh` to create `Lectures Assistant.app`

Users double-click the app, which starts the server and opens the browser automatically.

**Windows:**
Run `./package-windows.sh` to create `Lectures Assistant/`

Users double-click `Lectures Assistant.bat`, which starts the server and opens the browser automatically.

## Building Without Docker

If you prefer to build from source without Docker:

**Install dependencies**

Mac: `brew install ffmpeg poppler go node`
Linux: `sudo apt install ffmpeg poppler-utils golang nodejs npm`
Windows: Install Go, Node.js, ffmpeg, and poppler from their websites

**Build and run**

Run `./build.sh` to create native binaries in `server/dist/`
Run `./start.sh` (or `start.bat`) which auto-detects and uses your built binary

## For Developers

**Project structure**

- `server/` - Go backend server
- `website/` - SvelteKit frontend
- `Dockerfile` - Multi-stage build (frontend + backend + dependencies)

**Development mode**

Backend: `cd server && go run ./cmd/server -configuration ../configuration.yaml`
Frontend: `cd website && npm run dev`

**Production builds**

Native binaries: `./build.sh` (outputs to `server/dist/`)
Docker: `docker-compose up --build`
