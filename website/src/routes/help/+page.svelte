<script lang="ts">
  import Breadcrumb from "$lib/components/Breadcrumb.svelte";
  import { BookOpen, Upload, Cpu, FileText, MessageSquare, Download } from "lucide-svelte";
</script>

<Breadcrumb items={[{ label: "Features", active: true }]} />

<div class="row g-4">
  <div class="col-md-6">
    <div class="step-card bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <BookOpen size={16} class="text-orange" />
          <span class="header-text">1. Create a Subject</span>
        </div>
      </div>
      <div class="p-4">
        <p class="text-muted mb-2">
          Go to <strong>My Studies</strong> and create a subject for the course or topic you are studying — for example, "Biochemistry" or "Constitutional Law".
        </p>
        <p class="text-muted mb-0">
          Set the language so that transcripts and generated materials default to it automatically.
        </p>
      </div>
    </div>
  </div>

  <div class="col-md-6">
    <div class="step-card bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <Upload size={16} class="text-orange" />
          <span class="header-text">2. Add a Lesson</span>
        </div>
      </div>
      <div class="p-4">
        <p class="text-muted mb-2">
          Inside a subject, add a lesson and upload your files. You can mix recording files and reference documents in the same upload.
        </p>
        <ul class="text-muted mb-0 ps-3">
          <li class="mb-1"><strong>Recordings</strong> — mp4, mkv, mov, webm, mp3, wav, m4a, flac</li>
          <li><strong>Reference materials</strong> — pdf, pptx, docx</li>
        </ul>
      </div>
    </div>
  </div>

  <div class="col-md-6">
    <div class="step-card bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <Cpu size={16} class="text-orange" />
          <span class="header-text">3. Let AI Process</span>
        </div>
      </div>
      <div class="p-4">
        <p class="text-muted mb-2">
          Processing starts automatically. The AI transcribes your recordings into a full segmented transcript and extracts the content of every uploaded document.
        </p>
        <p class="text-muted mb-0">
          You can navigate away — jobs run in the background and you will see progress updates in real time when you return.
        </p>
      </div>
    </div>
  </div>

  <div class="col-md-6">
    <div class="step-card bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <FileText size={16} class="text-orange" />
          <span class="header-text">4. Review Your Lesson</span>
        </div>
      </div>
      <div class="p-4">
        <p class="text-muted mb-2">
          Open a lesson to access its Workspace. Three views are available:
        </p>
        <ul class="text-muted mb-0 ps-3">
          <li class="mb-1"><strong>Dialogue</strong> — the full transcript, segmented and readable, with the original audio inline.</li>
          <li class="mb-1"><strong>Reference Materials</strong> — browse and read uploaded documents page by page with extracted text.</li>
          <li><strong>Study Guide</strong> — generate a comprehensive guide grounded in both the transcript and your reference materials.</li>
        </ul>
      </div>
    </div>
  </div>

  <div class="col-md-6">
    <div class="step-card bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <MessageSquare size={16} class="text-orange" />
          <span class="header-text">5. Ask the Study Assistant</span>
        </div>
      </div>
      <div class="p-4">
        <p class="text-muted mb-2">
          Each subject has a dedicated <strong>Study Assistant</strong> — a chat interface with full context over every lesson, transcript, and document in that subject.
        </p>
        <p class="text-muted mb-0">
          Ask it to clarify a concept, find a specific topic across lectures, summarise a section, or compare ideas from different lessons. Citations link directly to the source passage.
        </p>
      </div>
    </div>
  </div>

  <div class="col-md-6">
    <div class="step-card bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <Download size={16} class="text-orange" />
          <span class="header-text">6. Export</span>
        </div>
      </div>
      <div class="p-4">
        <p class="text-muted mb-2">
          Any transcript, study guide, or reference document can be exported from its workspace using the download icon.
        </p>
        <ul class="text-muted mb-0 ps-3">
          <li class="mb-1"><strong>PDF with images</strong> — fully typeset via XeLaTeX, including embedded diagrams and QR codes.</li>
          <li class="mb-1"><strong>PDF (text only)</strong> — compact, no images.</li>
          <li><strong>Word (.docx)</strong> — editable document for further annotation.</li>
        </ul>
      </div>
    </div>
  </div>
</div>

<style lang="scss">
  .step-card {
    border-radius: var(--border-radius);
  }
</style>
