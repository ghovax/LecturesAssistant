<script lang="ts">
    import { onMount } from 'svelte';
    import { auth } from '$lib/auth.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import Modal from '$lib/components/Modal.svelte';
    import { X } from 'lucide-svelte';

    let showWelcomeModal = $state(false);

    onMount(async () => {
        await auth.check();
    });
</script>

<div class="d-none d-md-block">
    <div class="bg-white border mb-5">
        <div class="standard-header">
            <div class="header-title">
                <span class="header-glyph" lang="ja">迎</span>
                <span class="header-text">Welcome to Learning Assistant</span>
            </div>
        </div>
        <div class="p-4">
            <p class="lead fw-bold text-success mb-4">Transform your lessons into smart study materials. Let's learn!</p>

            <div class="prose">
                <p>Learning complex subjects takes a lot of practice, but this platform will take care of the heavy lifting for you.
                You can stop wasting time on transcribing and just focus on learning.</p>
                
                <p>Why not give it a try? Get your recordings, slides, or PDFs ready and click the button below to begin.</p>
            </div>
            
            <p class="text-center mt-4 mb-0">
                {#if !auth.user}
                    {#if !auth.initialized}
                        <a href="/setup" class="btn btn-success btn-lg px-5 rounded-0">Begin Learning</a>
                    {:else}
                        <a href="/login" class="btn btn-success btn-lg px-5 rounded-0">Begin Learning</a>
                    {/if}
                {:else}
                    <a href="/exams" class="btn btn-success btn-lg px-5 rounded-0">Open My Study Hub</a>
                {/if}
            </p>
        </div>
    </div>
</div>

<div class="d-block d-md-none mb-4">
    <p class="text-center">
        <button type="button" class="btn btn-primary rounded-0" onclick={() => showWelcomeModal = true}>
            About this Website
        </button>
    </p>

    <Modal 
        title="Welcome" 
        glyph="迎" 
        isOpen={showWelcomeModal} 
        onClose={() => showWelcomeModal = false}
    >
        <div class="prose">
            <p class="lead fw-bold text-success">Learning Assistant transforms your lessons into study materials.</p>
            <p>Generating study materials is easier than ever. Simply upload your recordings and reference files.</p>
        </div>
        
        <div class="text-center mt-4">
            {#if !auth.user}
                {#if !auth.initialized}
                    <a href="/setup" class="btn btn-success rounded-0 w-100 mb-2">Begin Learning</a>
                {:else}
                    <a href="/login" class="btn btn-success rounded-0 w-100 mb-2">Begin Learning</a>
                {/if}
            {:else}
                <a href="/exams" class="btn btn-success rounded-0 w-100 mb-2">Go to My Studies</a>
            {/if}
        </div>
    </Modal>
</div>

<div class="linkTiles tileSizeMd">
    <Tile href="/exams" icon="辞" title="My Studies">
        {#snippet description()}
            Access your subjects, lessons, and all generated learning materials.
        {/snippet}
    </Tile>

    <Tile href="/settings" icon="設" title="Preferences">
        {#snippet description()}
            Customize your language, AI models, and interface settings.
        {/snippet}
    </Tile>

    <Tile href="/help" icon="新" title="What's New">
        {#snippet description()}
            Discover the latest features and capabilities in your assistant.
        {/snippet}
    </Tile>

    <Tile href="/feedback" icon="談" title="Send Feedback">
        {#snippet description()}
            Have a suggestion or found a bug? We'd love to hear from you!
        {/snippet}
    </Tile>

    <Tile href="/support" icon="助" title="Support Us">
        {#snippet description()}
            Help keep this project alive and growing with a small contribution.
        {/snippet}
    </Tile>

    <Tile href="/credits" icon="謝" title="Credits">
        {#snippet description()}
            A special thanks to the people and projects that make this possible.
        {/snippet}
    </Tile>
</div>