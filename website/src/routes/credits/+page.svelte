<script lang="ts">
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Heart } from 'lucide-svelte';
</script>

<Breadcrumb items={[{ label: 'Credits', active: true }]} />

<h1>Credits</h1>

<div class="well text-center">
    <Heart class="text-danger mb-3" size={48} />
    <h3>Thank You</h3>
    <p class="lead">This Learning Assistant platform is powered by amazing services and open-source contributions.</p>
    
    <div class="row mt-5 text-start">
        <div class="col-md-6 mb-4">
            <h4>Intelligence Providers</h4>
            <p>Our sophisticated analysis and generation capabilities are made possible through:</p>
            <ul>
                <li><strong>OpenRouter</strong> - Access to world-class cloud AI models</li>
                <li><strong>Ollama</strong> - Efficient local inference for privacy and speed</li>
            </ul>
        </div>
        <div class="col-md-6 mb-4">
            <h4>Design Inspiration</h4>
            <p>The clean and intuitive user interface was inspired by the elegant simplicity of <strong>Kakimashou</strong>, focusing on direct access to complex educational tools.</p>
        </div>
    </div>

    <p class="mt-4 small text-muted">A special thanks to the countless open-source developers whose foundational tools make complex processing accessible to everyone.</p>
</div>