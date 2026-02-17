<script lang="ts">
  import Breadcrumb from "$lib/components/Breadcrumb.svelte";
  import {
    Zap,
    BookOpen,
    MessageSquare,
    ShieldCheck,
    Download,
    Search,
  } from "lucide-svelte";
</script>

<Breadcrumb items={[{ label: "Features", active: true }]} />

<div class="row">
  <div class="col-md-6 mb-4">
    <div class="bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Lesson Transcripts</span>
        </div>
      </div>
      <div class="p-4">
        <div class="d-flex align-items-start gap-3">
          <Zap class="text-orange mt-1 flex-shrink-0" size={24} />
          <p class="mb-0 text-muted">
            High-fidelity conversion of your lesson audio and video into
            structured, readable text. Our AI handles technical terminology and
            complex reasoning with extreme precision.
          </p>
        </div>
      </div>
    </div>
  </div>

  <div class="col-md-6 mb-4">
    <div class="bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Smart Study Aids</span>
        </div>
      </div>
      <div class="p-4">
        <div class="d-flex align-items-start gap-3">
          <BookOpen class="text-orange mt-1 flex-shrink-0" size={24} />
          <p class="mb-0 text-muted">
            Automatically generate comprehensive study guides deeply grounded in
            your specific lesson materials and reference files.
          </p>
        </div>
      </div>
    </div>
  </div>

  <div class="col-md-6 mb-4">
    <div class="bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Study Assistant</span>
        </div>
      </div>
      <div class="p-4">
        <div class="d-flex align-items-start gap-3">
          <MessageSquare class="text-orange mt-1 flex-shrink-0" size={24} />
          <p class="mb-0 text-muted">
            Interact with your materials through an intelligent chat interface.
            Ask questions, clarify concepts, and explore connections across all
            your lessons simultaneously.
          </p>
        </div>
      </div>
    </div>
  </div>

  <div class="col-md-6 mb-4">
    <div class="bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Deep Content Search</span>
        </div>
      </div>
      <div class="p-4">
        <div class="d-flex align-items-start gap-3">
          <Search class="text-orange mt-1 flex-shrink-0" size={24} />
          <p class="mb-0 text-muted">
            Locate specific topics, quotes, or data points instantly across
            thousands of pages of documents and hours of text with integrated
            global search.
          </p>
        </div>
      </div>
    </div>
  </div>

  <div class="col-md-6 mb-4">
    <div class="bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Professional Export</span>
        </div>
      </div>
      <div class="p-4">
        <div class="d-flex align-items-start gap-3">
          <Download class="text-orange mt-1 flex-shrink-0" size={24} />
          <p class="mb-0 text-muted">
            Export your generated materials into beautifully formatted PDF,
            Word, or Markdown documents, ready for offline study or printing.
          </p>
        </div>
      </div>
    </div>
  </div>

  <div class="col-md-6 mb-4">
    <div class="bg-white border h-100">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Local & Cloud AI</span>
        </div>
      </div>
      <div class="p-4">
        <div class="d-flex align-items-start gap-3">
          <ShieldCheck class="text-orange mt-1 flex-shrink-0" size={24} />
          <p class="mb-0 text-muted">
            Flexibility to use powerful cloud-based models or fully private,
            local AI services like Ollama, keeping your data under your control.
          </p>
        </div>
      </div>
    </div>
  </div>
</div>
