<script lang="ts">
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Heart } from 'lucide-svelte';
</script>

<Breadcrumb items={[{ label: 'Credits', active: true }]} />

<div class="bg-white border mb-3">
    <div class="standard-header">
        <div class="header-title">
            <span class="header-text">System Credits</span>
        </div>
    </div>

    <div class="p-5">
        <div class="text-center mb-5">
            <Heart class="text-orange mb-4" size={48} />
            <h1 class="display-6 fw-bold">Acknowledgments</h1>
            <p class="lead text-muted">This Learning Assistant platform is built upon a foundation of world-class open-source technologies and AI services.</p>
        </div>
        
        <div class="row g-4">
            <div class="col-md-6">
                <div class="p-4 border bg-light bg-opacity-10 h-100">
                    <h4 class="fw-bold mb-3 d-flex align-items-center">
                        AI Foundations
                    </h4>
                    <p>Our analysis and generation capabilities are powered by a hybrid approach to inference:</p>
                    <ul class="mb-0">
                        <li class="mb-2"><strong>OpenRouter</strong> - Providing seamless access to frontier cloud models.</li>
                        <li><strong>Ollama</strong> - Enabling private, secure local inference for sensitive content.</li>
                    </ul>
                </div>
            </div>
            
            <div class="col-md-6">
                <div class="p-4 border bg-light bg-opacity-10 h-100">
                    <h4 class="fw-bold mb-3 d-flex align-items-center">
                        Design Inspiration
                    </h4>
                    <p>The visual language and core user experience of this project were deeply inspired by minimalist "craft" principles.</p>
                    <p class="mb-0">We aim to provide a warm, direct interface applied to the modern landscape of AI-assisted education.</p>
                </div>
            </div>
        </div>

        <div class="mt-5 text-center">
            <p class="small text-muted mb-0">Special thanks to the Svelte, Vite, and Bootstrap communities for the tools that make development a joy.</p>
        </div>
    </div>
</div>
