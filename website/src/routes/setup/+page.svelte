<script lang="ts">
  import { api } from "$lib/api/client";
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";
  import { auth } from "$lib/auth.svelte";
  import { notifications } from "$lib/stores/notifications.svelte";
  import Breadcrumb from "$lib/components/Breadcrumb.svelte";
  import { Upload, Database } from "lucide-svelte";

  let username = $state("");
  let password = $state("");
  let apiKey = $state("");
  let error = $state("");
  let loading = $state(false);
  let restoring = $state(false);

  onMount(() => {
    // If setup is already completed, redirect to home
    if (auth.initialized) {
      goto("/");
    }
  });

  async function handleSetup() {
    loading = true;
    error = "";
    try {
      const data = await api.setup({
        username,
        password,
        openrouter_api_key: apiKey,
      });
      // The setup endpoint returns a token, we should use it
      localStorage.setItem("session_token", data.token);
      goto("/exams");
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function handleRestore(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    restoring = true;
    error = "";
    try {
      await api.restoreDatabase(input.files[0]);
      notifications.success("Workspace restored successfully.");
      // Redirect to login since users now exist
      goto("/login");
    } catch (e: any) {
      error = e.message;
      notifications.error(e.message);
    } finally {
      restoring = false;
    }
  }
</script>

<Breadcrumb items={[{ label: "Get Started", active: true }]} />

<div class="row justify-content-center">
  <div class="col-lg-8">
    <div class="bg-white border mb-4">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Let's Get Started</span>
        </div>
      </div>

      <div class="p-4">
        <div class="prose mb-4">
          <p class="mb-0">
            Welcome! Create your account and provide your API credentials to
            begin your learning journey.
          </p>
        </div>

        <form
          onsubmit={(e) => {
            e.preventDefault();
            handleSetup();
          }}
        >
          {#if error && !restoring}
            <div
              class="alert alert-danger mb-4 rounded-0 border-danger bg-danger bg-opacity-10 text-danger"
            >
              {error}
            </div>
          {/if}

          <div class="row">
            <div class="col-md-6 mb-4">
              <label for="username" class="cozy-label">Admin Username</label>
              <input
                type="text"
                id="username"
                class="form-control cozy-input"
                bind:value={username}
                required
              />
            </div>

            <div class="col-md-6 mb-4">
              <label for="password" class="cozy-label">Admin Password</label>
              <input
                type="password"
                id="password"
                class="form-control cozy-input"
                bind:value={password}
                required
                minlength="8"
              />
              <small class="form-text text-muted mt-2 d-block"
                >Minimum 8 characters.</small
              >
            </div>
          </div>

          <div class="mb-4">
            <label for="apiKey" class="cozy-label">OpenRouter API Key</label>
            <input
              type="password"
              id="apiKey"
              class="form-control cozy-input"
              bind:value={apiKey}
              required
            />
            <small class="form-text text-muted mt-2 d-block"
              >This key is required for all AI-powered transcription and
              generation features.</small
            >
          </div>

          <div class="text-center">
            <button
              type="submit"
              class="btn btn-success px-5 btn-lg rounded-0 w-100"
              disabled={loading || restoring}
            >
              {#if loading}
                <span
                  class="spinner-border spinner-border-sm me-2"
                  role="status"
                ></span>
              {/if}
              Create My Account
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Restore Section -->
    <div class="bg-white border mb-3">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Already have a workspace?</span>
        </div>
      </div>
      <div class="p-4">
        <p class="text-muted small mb-4">
          If you have a previously exported database file, you can restore it
          here to recover your subjects, lessons, and configurations.
        </p>

        <div class="text-center">
          <input
            type="file"
            id="restore-file"
            class="d-none"
            accept=".db"
            onchange={handleRestore}
            disabled={loading || restoring}
          />
          <label
            for="restore-file"
            class="btn btn-outline-primary px-5 rounded-0 w-100 d-flex align-items-center justify-content-center gap-2"
            class:disabled={loading || restoring}
          >
            {#if restoring}
              <span class="spinner-border spinner-border-sm" role="status"
              ></span>
              Restoring Workspace...
            {:else}
              <Database size={18} />
              Import Existing Workspace
            {/if}
          </label>
        </div>
      </div>
    </div>
  </div>
</div>
