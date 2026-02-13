<script lang="ts">
    import { onMount } from 'svelte';
    import { auth } from '$lib/auth.svelte';
    import { goto } from '$app/navigation';
    import Tile from '$lib/components/Tile.svelte';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';

    async function handleLogout() {
        await auth.logout();
        goto('/');
    }

    onMount(async () => {
        await auth.check();
    });
</script>

<Breadcrumb items={[]} />

<div class="bg-white border mb-3">
    <div class="standard-header">
        <div class="header-title">
            <span class="header-glyph" lang="ja">辞</span>
            <span class="header-text">Workspace</span>
        </div>
    </div>
    <div class="p-2">
        <div class="linkTiles">
            <Tile href="/exams" icon="辞" title="My Studies">
                {#snippet description()}
                    Access subjects, lessons, and all generated materials.
                {/snippet}
            </Tile>
            <Tile href="/settings" icon="設" title="Preferences">
                {#snippet description()}
                    Customize language, AI models, and interface.
                {/snippet}
            </Tile>
        </div>
    </div>
</div>

<div class="row g-3">
    <div class="col-lg-6">
        <div class="bg-white border mb-3 h-100">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-glyph" lang="ja">人</span>
                    <span class="header-text">Account & Session</span>
                </div>
            </div>
            <div class="p-2">
                <div class="linkTiles">
                    {#if !auth.user}
                        <Tile href="/login" icon="入" title="Sign In">
                            {#snippet description()}
                                Access your personal study hub.
                            {/snippet}
                        </Tile>
                    {:else}
                        <Tile href="/profile" icon="人" title="My Profile">
                            {#snippet description()}
                                View and manage your account details.
                            {/snippet}
                        </Tile>
                        <Tile onclick={handleLogout} icon="出" title="Logout">
                            {#snippet description()}
                                Securely sign out of your current session.
                            {/snippet}
                        </Tile>
                    {/if}
                </div>
            </div>
        </div>
    </div>
    <div class="col-lg-6">
        <div class="bg-white border mb-3 h-100">
            <div class="standard-header">
                <div class="header-title">
                    <span class="header-glyph" lang="ja">新</span>
                    <span class="header-text">Resources</span>
                </div>
            </div>
            <div class="p-2">
                <div class="linkTiles">
                    <Tile href="/help" icon="新" title="Help Guide">
                        {#snippet description()}
                            How to use the assistant effectively.
                        {/snippet}
                    </Tile>
                    <Tile href="/credits" icon="謝" title="Credits">
                        {#snippet description()}
                            System acknowledgments and contributors.
                        {/snippet}
                    </Tile>
                </div>
            </div>
        </div>
    </div>
</div>
