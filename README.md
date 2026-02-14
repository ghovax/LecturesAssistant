# Lectures Assistant

Lectures Assistant is a high-fidelity, AI-powered platform designed to transform lecture recordings and reference materials into structured, pedagogical study documents. By leveraging frontier LLMs via OpenRouter or secure local models via Ollama, it provides an immersive and organized learning experience.

## ✨ Core Features

- **Multi-modal AI Ingestion**: High-precision transcription of audio/video recordings and intelligent interpretation of PDF, PPTX, and DOCX documents.
- **Smart Study Aids**: Automatically generate comprehensive study guides, flashcard sets, and multiple-choice quizzes grounded deeply in your materials.
- **AI Reading Assistant**: An integrated chat interface that lets you ask questions, clarify concepts, and explore connections across all your lessons simultaneously.
- **Professional Exports**: Export your materials to beautifully formatted PDF (via XeLaTeX), Word (Docx), or Markdown, complete with embedded cited images and QR codes for easy sharing.
- **Cost Tracking**: Transparent monitoring of token usage and estimated USD costs for every AI-powered operation.
- **Local & Cloud Privacy**: Choose between powerful cloud models (Claude, GPT, Gemini) or fully private local inference using Ollama.

---

## 🚀 Getting Started

The latest pre-built packages for macOS and Windows are available in the **[Releases](https://github.com/user/LecturesAssistant/releases)** section.

### Packaged Applications (Recommended)

#### macOS

1. Download the `Lectures.Assistant.mac.zip` from Releases.
2. Extract the archive and move `Lectures Assistant.app` to your Applications folder.
3. Double-click the app to start. It will open a terminal window and your default browser at `http://localhost:3000`.

#### Windows

1. Download the `Lectures.Assistant.windows.zip` from Releases.
2. Extract the folder.
3. Double-click `start.bat` to launch the application.

### Running with Docker

If you have Docker installed, you can run the entire environment (including all system dependencies like FFmpeg and XeLaTeX) with a single command:

```bash
docker-compose up --build -d
```

Access the interface at `http://localhost:3000`.

---

## 🛠️ Configuration

On your first run, you will be guided through a **Setup** process:

1. **Create an Admin Account**: Secure your local instance.
2. **AI Provider**:
   - **OpenRouter (Cloud)**: Provide your API key from [openrouter.ai](https://openrouter.ai/).
   - **Ollama (Local)**: Ensure Ollama is running locally.
3. **Language**: Set your primary study language (transcripts and guides will default to this).

---

## 🏗️ Building from Source

### Prerequisites

- **Go** (1.23+)
- **Node.js** (20+) & **npm**
- **System Tools**: FFmpeg, Ghostscript, Pandoc, and Tectonic (for PDF exports).

### Build Scripts

We provide automated build scripts in the root directory:

- **macOS**: `./build-mac.sh` (Generates a native `.app` bundle with icons)
- **Windows**: `./build-windows.sh` (Generates a distribution folder with `.exe` and `start.bat`)

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

---

## 📂 Project Structure

- `server/`: Go backend handles API, job queue, LLM routing, and file processing.
- `website/`: SvelteKit frontend providing a minimalist, "craft" inspired user interface.
- `prompts/`: A centralized repository of system prompts used for AI orchestration.

---

## 📝 Troubleshooting

**Port Conflicts:**
By default, the server uses port `3000`. If this port is occupied, update `server.port` in `configuration.yaml`.

**Dependency Errors:**
If exports fail, ensure `pandoc` and `tectonic` are available in your system PATH. Packaged apps and Docker versions include these automatically.

---

Built with focus and care for students who value clarity and depth in their learning journey.
