<script lang="ts">
  import { auth } from "$lib/auth.svelte";
  import {
    Book,
    Settings,
    User,
    LogIn,
    LogOut,
    Sparkles,
    HelpCircle,
  } from "lucide-svelte";
  import { goto } from "$app/navigation";
  let isMenuOpen = $state(false);
  async function handleLogout() {
    isMenuOpen = false;
    await auth.logout();
    goto("/");
  }
</script>

<nav class="navbar navbar-expand-md cozy-navbar p-0">
  <!-- Brand and toggle get grouped for better mobile display -->
  <a class="navbar-brand px-0 py-0 me-4" href="/">
    Lectures<span class="text-orange">Assistant</span>
  </a>
  <button
    class="navbar-toggler border-0 shadow-none"
    type="button"
    onclick={() => (isMenuOpen = !isMenuOpen)}
    aria-controls="navbarSupportedContent"
    aria-expanded={isMenuOpen}
    aria-label="Toggle navigation"
  >
    <span class="navbar-toggler-icon"></span>
  </button>
  <!-- Collect the nav links, forms, and other content for toggling -->
  <div
    class="collapse navbar-collapse {isMenuOpen ? 'show' : ''}"
    id="navbarSupportedContent"
  >
    <ul class="navbar-nav me-auto">
      <li class="nav-item">
        <a
          class="nav-link px-2"
          href="/exams"
          onclick={() => (isMenuOpen = false)}
        >
          <span class="nav-icon" aria-hidden="true"><Book size={16} /></span>
          <span class="nav-text">My Studies</span>
        </a>
      </li>
      <li class="nav-item">
        <a
          class="nav-link px-2"
          href="/help"
          onclick={() => (isMenuOpen = false)}
        >
          <span class="nav-icon" aria-hidden="true"
            ><HelpCircle size={16} /></span
          >
          <span class="nav-text">Guide</span>
        </a>
      </li>
    </ul>
    <ul id="rightNav" class="navbar-nav ms-auto">
      <li class="nav-item">
        <a
          class="nav-link px-2"
          href="/settings"
          onclick={() => (isMenuOpen = false)}
        >
          <span class="nav-icon" aria-hidden="true"><Settings size={16} /></span
          >
          <span class="nav-text">Preferences</span>
        </a>
      </li>
      {#if auth.user}
        <li class="nav-item">
          <button
            class="nav-link btn btn-link border-0 shadow-none px-2 w-100 justify-content-start"
            onclick={handleLogout}
          >
            <span class="nav-icon" aria-hidden="true"><LogOut size={16} /></span
            >
            <span class="nav-text">Logout</span>
          </button>
        </li>
      {:else}
        <li class="nav-item">
          <a
            class="nav-link px-2"
            href="/login"
            onclick={() => (isMenuOpen = false)}
          >
            <span class="nav-icon" aria-hidden="true"><LogIn size={16} /></span>
            <span class="nav-text">Login</span>
          </a>
        </li>
      {/if}
    </ul>
  </div>
</nav>

<style lang="scss">
  .cozy-navbar {
    font-family: var(--font-primary);
    background: transparent;
    border-bottom: 1px solid var(--gray-300);
    min-height: 60px;
  }

  .navbar-brand {
    font-size: 20px;
    font-weight: 500;
    color: var(--gray-900) !important;
    letter-spacing: -0.02em;
    display: flex;
    align-items: center;
    height: 60px;
    margin: 0;

    .text-orange {
      color: var(--orange);
      font-weight: 600;
    }
  }

  .nav-link {
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--gray-600) !important;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all 0.2s ease;
    height: 60px;
    border-bottom: 2px solid transparent;

    &:hover {
      color: var(--gray-900) !important;
      border-bottom-color: var(--orange);
    }

    .nav-icon {
      display: flex;
      align-items: center;
      color: var(--gray-400);
      transition: color 0.2s ease;
    }

    &:hover .nav-icon {
      color: var(--orange);
    }
  }

  @media (max-width: 768px) {
    .navbar-collapse {
      padding: 12px 0 20px 0;
    }

    .nav-link {
      height: auto;
      padding: 14px 0;

      &:hover {
        padding-left: 8px;
      }
    }

    #rightNav {
      margin-top: 12px;
      padding-top: 12px;
      border-top: 1px solid var(--gray-300);
    }
  }
</style>
