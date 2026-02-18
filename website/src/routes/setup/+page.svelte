<script lang="ts">
  import { api } from "$lib/api/client";
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";
  import { auth } from "$lib/auth.svelte";
  import Breadcrumb from "$lib/components/Breadcrumb.svelte";

  let username = $state("");
  let password = $state("");
  let apiKey = $state("");
  let error = $state("");
  let loading = $state(false);

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
</script>

<Breadcrumb items={[{ label: "Get Started", active: true }]} />

<div class="row justify-content-center">
  <div class="col-lg-8">
    <div class="bg-white border mb-3">
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
          {#if error}
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
              disabled={loading}
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

    <style lang="scss">
    </style>
  </div>
</div>
