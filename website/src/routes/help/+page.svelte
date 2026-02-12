<script lang="ts">
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Zap, BookOpen, MessageSquare, ShieldCheck, Download, Search } from 'lucide-svelte';
</script>

<Breadcrumb items={[{ label: 'Features', active: true }]} />

<div class="bg-white border mb-5">
    <div class="standard-header">
        <div class="header-title">
            <span class="header-glyph" lang="ja">機</span>
            <span class="header-text">Features & Capabilities</span>
        </div>
    </div>

    <div class="p-4">
        <div class="row">
            <div class="col-md-6 mb-4">
                <div class="well h-100">
                    <div class="d-flex align-items-center mb-3">
                        <Zap class="text-primary me-2" size={24} />
                        <h4 class="m-0">Lesson Transcripts</h4>
                    </div>
                    <p>High-fidelity conversion of your lesson audio and video into structured, readable text. Our AI handles technical terminology and complex reasoning with extreme precision.</p>
                </div>
            </div>

            <div class="col-md-6 mb-4">
                <div class="well h-100">
                    <div class="d-flex align-items-center mb-3">
                        <BookOpen class="text-primary me-2" size={24} />
                        <h4 class="m-0">Smart Study Aids</h4>
                    </div>
                    <p>Automatically generate comprehensive study guides, flashcards, and practice quizzes. Every resource is deeply grounded in your specific lesson materials and reference files.</p>
                </div>
            </div>

            <div class="col-md-6 mb-4">
                <div class="well h-100">
                    <div class="d-flex align-items-center mb-3">
                        <MessageSquare class="text-primary me-2" size={24} />
                        <h4 class="m-0">Study Assistant</h4>
                    </div>
                    <p>Interact with your materials through an intelligent chat interface. Ask questions, clarify concepts, and explore connections across all your lessons simultaneously.</p>
                </div>
            </div>

            <div class="col-md-6 mb-4">
                <div class="well h-100">
                    <div class="d-flex align-items-center mb-3">
                        <Search class="text-primary me-2" size={24} />
                        <h4 class="m-0">Deep Content Search</h4>
                    </div>
                    <p>Locate specific topics, quotes, or data points instantly across thousands of pages of documents and hours of text with integrated global search.</p>
                </div>
            </div>

            <div class="col-md-6 mb-4">
                <div class="well h-100">
                    <div class="d-flex align-items-center mb-3">
                        <Download class="text-primary me-2" size={24} />
                        <h4 class="m-0">Professional Export</h4>
                    </div>
                    <p>Export your generated materials into beautifully formatted PDF, Word, or Markdown documents, ready for offline study or printing.</p>
                </div>
            </div>

            <div class="col-md-6 mb-4">
                <div class="well h-100">
                    <div class="d-flex align-items-center mb-3">
                        <ShieldCheck class="text-primary me-2" size={24} />
                        <h4 class="m-0">Local & Cloud AI</h4>
                    </div>
                    <p>Flexibility to use powerful cloud-based models or fully private, local AI services like Ollama, keeping your data under your control.</p>
                </div>
            </div>
        </div>
    </div>
</div>
