# Lectures Assistant

Lectures Assistant is a high-fidelity, AI-powered platform designed to transform lecture recordings and reference materials into structured, pedagogical study documents. By leveraging frontier LLMs via OpenRouter or secure local models via Ollama, it provides an immersive and organized learning experience.

## ✨ Core Features

- **Multi-modal AI Ingestion**: High-precision transcription of audio/video recordings and intelligent interpretation of PDF, PPTX, and DOCX documents.
- **Smart Study Aids**: Automatically generate comprehensive study guides, flashcard sets, and multiple-choice quizzes grounded deeply in your materials.
- **AI Reading Assistant**: An integrated chat interface that lets you ask questions, clarify concepts, and explore connections across all your lessons simultaneously.
- **Professional Exports**: Export your materials to beautifully formatted PDF (via XeLaTeX), Word (Docx), or Markdown, complete with embedded cited images and QR codes for easy sharing.
- **Cost Tracking**: Transparent monitoring of token usage and estimated USD costs for every AI-powered operation.
- **Local & Cloud Privacy**: Choose between powerful cloud models (Claude, GPT, Gemini) or fully private local inference using Ollama.

## 🚀 Quick Start

### For Non-Technical Users (Recommended)

The easiest way to run Lectures Assistant is with a single Docker command:

```bash
docker run -d --name lectures-assistant -p 3000:3000 giovanni653/lectures-assistant:latest
```

Then open [http://localhost:3000](http://localhost:3000) in your browser.

**That's it!** The application will:
- Download automatically (~900 MB)
- Start running in the background
- Store all your data safely on your computer
- Be available at http://localhost:3000

**Useful commands:**
- Stop: `docker stop lectures-assistant`
- Start again: `docker start lectures-assistant`
- View logs: `docker logs lectures-assistant`
- Remove completely: `docker rm -f lectures-assistant`

---

### For Developers

If you prefer using Docker Compose or want to build from source:

**Using Docker Compose:**
```bash
docker-compose up -d
```

**Build from source:** See the [Development](#-development) section below.

## 🛠️ Configuration

On your first run, you will be guided through a **Setup** process:

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

- `server/`: Go backend handles API, job queue, LLM routing, and file processing.
- `website/`: SvelteKit frontend providing a minimalist, "craft" inspired user interface.
- `prompts/`: A centralized repository of system prompts used for AI orchestration.

## 📝 Troubleshooting

**Port Conflicts:**
By default, the server uses port `3000`. If this port is occupied, update `server.port` in `configuration.yaml`.

**Dependency Errors:**
If exports fail, ensure `pandoc` and `tectonic` are available in your system PATH. Docker versions include these automatically.

Built with focus and care for students who value clarity and depth in their learning journey.
