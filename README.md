# Lectures Assistant

Lectures Assistant is a high-fidelity, AI-powered platform designed to transform lecture recordings and reference materials into structured, pedagogical study documents. By leveraging frontier LLMs via OpenRouter or secure local models via Ollama, it provides an immersive and organized learning experience.

## ✨ Core Features

- **Multi-modal AI Ingestion**: High-precision transcription of audio/video recordings and intelligent interpretation of PDF, PPTX, and DOCX documents.
- **Smart Study Aids**: Automatically generate comprehensive study guides, flashcard sets, and multiple-choice quizzes grounded deeply in your materials.
- **AI Reading Assistant**: An integrated chat interface that lets you ask questions, clarify concepts, and explore connections across all your lessons simultaneously.
- **Professional Exports**: Export your materials to beautifully formatted PDF (via XeLaTeX), Word (Docx), or Markdown, complete with embedded cited images and QR codes for easy sharing.
- **Cost Tracking**: Transparent monitoring of token usage and estimated USD costs for every AI-powered operation.
- **Local & Cloud Privacy**: Choose between powerful cloud models or fully private local inference using Ollama.

## 🚀 Quick Start

Lectures Assistant runs entirely through Docker. Make sure you have [Docker Desktop](https://www.docker.com/products/docker-desktop/) installed, then run:

```bash
docker run -d --name lectures-assistant -p 3000:3000 giovanni653/lectures-assistant:latest
```

Then open [http://localhost:3000](http://localhost:3000) in your browser.

**Useful commands:**

```bash
docker stop lectures-assistant    # Stop
docker start lectures-assistant   # Start again
docker rm -f lectures-assistant   # Remove
docker logs -f lectures-assistant # View logs
```

## ⚙️ Configuration

On your first run you will be guided through a setup process:

1. **Create an Admin Account**: Secure your local instance.
2. **AI Provider**:
   - **OpenRouter (Cloud)**: Provide your API key from [openrouter.ai](https://openrouter.ai/).
   - **Ollama (Local)**: Ensure Ollama is running locally.
3. **Language**: Set your primary study language (transcripts and guides will default to this).

## 🏗️ Development

### Prerequisites

- **Go** (1.24+)
- **Node.js** (20+) & **npm**
- **System Tools**: FFmpeg, Ghostscript, Pandoc, and Tectonic (for PDF exports).

### Manual Development Flow

**Backend:**

```bash
cd server
go run ./cmd/server -configuration ../configuration.yaml
```

**Frontend:**

```bash
cd website
npm install
npm run dev
```

## 📂 Project Structure

- `server/`: Go backend handling the API, job queue, LLM routing, and file processing.
- `website/`: SvelteKit frontend providing a minimalist, craft-inspired user interface.
- `prompts/`: Centralized repository of system prompts used for AI orchestration.

## 📝 Troubleshooting

**Port Conflicts:**
By default the server uses port `3000`. Change the `-p` flag in the Docker command if this port is occupied (e.g. `-p 8080:3000`).

**Export Failures:**
The Docker image includes all required dependencies (Pandoc, Tectonic). If running outside Docker, ensure these are available in your system PATH.

---

Built with focus and care for students who value clarity and depth in their learning journey.
