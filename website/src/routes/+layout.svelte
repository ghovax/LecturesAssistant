<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/state";
  import { goto } from "$app/navigation";
  import { auth } from "$lib/auth.svelte";
  import "../app.scss";
  import Navbar from "$lib/components/Navbar.svelte";
  import NotificationBanner from "$lib/components/NotificationBanner.svelte";

  let { children } = $props();

  onMount(async () => {
    // @ts-ignore
    await import("bootstrap");

    if (auth.loading) {
      await auth.check();
    }

    // If not initialized and not already on the setup page, force redirect to setup
    if (!auth.initialized && page.url.pathname !== "/setup") {
      goto("/setup");
    }
  });
</script>

<div class="main-layout container-xl shadow-none">
  <Navbar />
  <NotificationBanner />
  <div class="content-wrapper">
    <main class="content">
      {@render children()}
    </main>
  </div>
</div>

<style lang="scss">
  :global(html, body) {
    height: 100%;
    margin: 0;
    background: #f5f3f0 !important;
    font-family: var(--font-primary) !important;
    color: #292524 !important;
  }

  .main-layout {
    background: transparent !important;
    box-shadow: none !important;
    border: none !important;
    max-width: 1300px !important; /* Increased width */
    margin: 0 auto;
    padding: 0 20px;
  }

  /* Override Bootstrap container padding to match navbar */
  .container-xl {
    padding-left: 0 !important;
    padding-right: 0 !important;
  }

  .content-wrapper {
    width: 100%;
  }

  .content {
    padding: 0 20px 80px;
  }
</style>
