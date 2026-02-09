<script lang="ts">
    import { auth } from '$lib/auth.svelte';
    import { Book, Activity, Settings, User } from 'lucide-svelte';

    let isMenuOpen = $state(false);
</script>

<nav class="navbar navbar-expand-md navbar-dark bg-dark p-0">
    <!-- Brand and toggle get grouped for better mobile display -->
    <a class="navbar-brand px-3 py-0 mr-0" href="/">
        <span lang="ja" aria-hidden="true">書</span> Learning Assistant
    </a>
    <button 
        class="navbar-toggler" 
        type="button" 
        onclick={() => isMenuOpen = !isMenuOpen}
        aria-controls="navbarSupportedContent" 
        aria-expanded={isMenuOpen} 
        aria-label="Toggle navigation"
    >
        <span class="navbar-toggler-icon"></span>
    </button>

    <!-- Collect the nav links, forms, and other content for toggling -->
    <div class="collapse navbar-collapse pl-3 pl-md-0 {isMenuOpen ? 'show' : ''}" id="navbarSupportedContent">
        <ul class="navbar-nav mr-auto">
            <li class="nav-item">
                <a class="nav-link" href="/exams" onclick={() => isMenuOpen = false}>
                    <span class="glyphicon" aria-hidden="true"><Book size={16} strokeWidth={3} /></span> My Studies
                </a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/jobs" onclick={() => isMenuOpen = false}>
                    <span class="glyphicon" aria-hidden="true"><Activity size={16} strokeWidth={3} /></span> Task Progress
                </a>
            </li>
        </ul>

        <ul id="rightNav" class="navbar-nav mr-3">
            <li class="nav-item">
                <a class="nav-link" href="/settings" onclick={() => isMenuOpen = false}>
                    <span class="glyphicon" aria-hidden="true"><Settings size={16} strokeWidth={3} /></span> Preferences
                </a>
            </li>
            {#if auth.user}
                <li class="nav-item">
                    <a class="nav-link" href="/profile" onclick={() => isMenuOpen = false}>
                        <span class="glyphicon" aria-hidden="true"><User size={16} strokeWidth={3} /></span> Profile
                    </a>
                </li>
            {/if}
        </ul>
    </div>
</nav>
