<script lang="ts">
    import { onMount } from 'svelte';
    import { auth } from '$lib/auth.svelte';
    import Tile from '$lib/components/Tile.svelte';
    import { X } from 'lucide-svelte';

    let showWelcomeModal = $state(false);

    onMount(async () => {
        await auth.check();
    });
</script>

<div class="d-none d-md-block">
    <div class="welcome-message">
        <h1>Welcome to Learning Assistant</h1>
        <p class="lead">Transform your lessons into smart study materials. Let's learn!</p>
    </div>

    <p>Learning complex subjects takes a lot of practice, but this platform will take care of the heavy lifting for you.
    You can stop wasting time on transcribing and just focus on learning.</p>
    
    <p>Why not give it a try? Get your recordings, slides, or PDFs ready and click the button below to begin.</p>
    
    <p class="text-center mt-4">
        {#if !auth.user}
            {#if !auth.initialized}
                <a href="/setup" class="btn btn-success btn-lg">Begin Learning</a>
            {:else}
                <a href="/login" class="btn btn-success btn-lg">Begin Learning</a>
            {/if}
        {:else}
            <a href="/exams" class="btn btn-success btn-lg">Open My Study Hub</a>
        {/if}
    </p>
</div>

<div class="d-block d-md-none">
    <p class="text-center">
        <button type="button" class="btn btn-primary" onclick={() => showWelcomeModal = true}>
            About this Website
        </button>
    </p>

    {#if showWelcomeModal}
        <div class="modal fade show d-block" tabindex="-1" role="dialog" style="background: rgba(0,0,0,0.5)">
            <div class="modal-dialog" role="document">
                <div class="modal-content">
                    <div class="modal-body position-relative">
                        <button type="button" class="close float-end border-0 bg-transparent" onclick={() => showWelcomeModal = false} aria-label="Close">
                            <span aria-hidden="true" style="font-size: 2rem;">×</span>
                        </button>
                        <div class="welcome-message">
                            <h1>Welcome</h1>
                            <p class="lead">Learning Assistant transforms your lessons into study materials.</p>
                        </div>

                        <p>Generating study materials is easier than ever. Simply upload your recordings and reference files.</p>
                        
                        <p class="text-center">
                            {#if !auth.user}
                                {#if !auth.initialized}
                                    <a href="/setup" class="btn btn-success btn-lg">Begin Learning</a>
                                {:else}
                                    <a href="/login" class="btn btn-success btn-lg">Begin Learning</a>
                                {/if}
                            {:else}
                                <a href="/exams" class="btn btn-success btn-lg">Go to My Studies</a>
                            {/if}
                        </p>
                    </div>
                </div>
            </div>
        </div>
    {/if}
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