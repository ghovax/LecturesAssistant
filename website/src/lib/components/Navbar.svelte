<script lang="ts">
    import { auth } from '$lib/auth.svelte';
    import { Book, Activity, Settings, User, LogIn, LogOut } from 'lucide-svelte';
    import { goto } from '$app/navigation';
    let isMenuOpen = $state(false);
    async function handleLogout() {
        isMenuOpen = false;
        await auth.logout();
        goto('/');
    }
</script>
<nav class="navbar navbar-expand-md navbar-dark bg-dark p-0 border-bottom border-dark-subtle">
    <!-- Brand and toggle get grouped for better mobile display -->
    <a class="navbar-brand px-3 py-0 me-0" href="/">
        <span lang="ja" aria-hidden="true">書</span> Assistant
    </a>
    <button 
        class="navbar-toggler border-0 shadow-none" 
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
        <ul class="navbar-nav me-auto">
            <li class="nav-item">
                <a class="nav-link px-3" href="/exams" onclick={() => isMenuOpen = false}>
                    <span class="glyphicon" aria-hidden="true"><Book size={14} strokeWidth={3} /></span> 
                    <span class="no-shift-bold" data-text="My Studies"><span>My Studies</span></span>
                </a>
            </li>
        </ul>
        <ul id="rightNav" class="navbar-nav me-3">
            <li class="nav-item">
                <a class="nav-link px-3" href="/settings" onclick={() => isMenuOpen = false}>
                    <span class="glyphicon" aria-hidden="true"><Settings size={14} strokeWidth={3} /></span> 
                    <span class="no-shift-bold" data-text="Preferences"><span>Preferences</span></span>
                </a>
            </li>
            {#if auth.user}
                <li class="nav-item">
                    <a class="nav-link px-3" href="/profile" onclick={() => isMenuOpen = false}>
                        <span class="glyphicon" aria-hidden="true"><User size={14} strokeWidth={3} /></span> 
                        <span class="no-shift-bold" data-text="Profile"><span>Profile</span></span>
                    </a>
                </li>
                <li class="nav-item">
                    <button class="nav-link btn btn-link border-0 shadow-none px-3 w-100 justify-content-start" onclick={handleLogout}>
                        <span class="glyphicon" aria-hidden="true"><LogOut size={14} strokeWidth={3} /></span> 
                        <span class="no-shift-bold" data-text="Logout"><span>Logout</span></span>
                    </button>
                </li>
            {:else}
                <li class="nav-item">
                    <a class="nav-link px-3" href="/login" onclick={() => isMenuOpen = false}>
                        <span class="glyphicon" aria-hidden="true"><LogIn size={14} strokeWidth={3} /></span> 
                        <span class="no-shift-bold" data-text="Login"><span>Login</span></span>
                    </a>
                </li>
            {/if}
        </ul>
    </div>
</nav>
<style lang="scss">
    .nav-link {
        transition: color 0.15s ease, background-color 0.15s ease;
        &:hover {
            background-color: rgba(255, 255, 255, 0.05);
            color: #fff !important;
            .no-shift-bold span {
                font-weight: bold;
            }
        }
    }
    .no-shift-bold {
        display: inline-grid;
        text-align: left;
    }
    .no-shift-bold::after {
        content: attr(data-text);
        grid-area: 1 / 1;
        font-weight: bold;
        visibility: hidden;
        height: 0;
    }
    .no-shift-bold > span {
        grid-area: 1 / 1;
    }
</style>
