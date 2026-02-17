<script lang="ts">
  import { api } from "$lib/api/client";
  import { auth } from "$lib/auth.svelte";
  import { goto } from "$app/navigation";
  import Breadcrumb from "$lib/components/Breadcrumb.svelte";

  let username = $state("");
  let password = $state("");
  let error = $state("");
  let loading = $state(false);

  async function handleLogin() {
    loading = true;
    error = "";
    try {
      await api.login({ username, password });
      await auth.check();
      goto("/exams");
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }
</script>

<Breadcrumb items={[{ label: "Login", active: true }]} />

<div class="row justify-content-center">
  <div class="col-lg-6">
    <div class="bg-white border mb-3">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Login</span>
        </div>
      </div>

      <div class="p-4">
        <form
          onsubmit={(e) => {
            e.preventDefault();
            handleLogin();
          }}
        >
          {#if error}
            <div
              class="alert alert-danger mb-4 rounded-0 border-danger bg-danger bg-opacity-10 text-danger"
            >
              {error}
            </div>
          {/if}

          <div class="mb-4">
            <label for="username" class="cozy-label">Username</label>
            <input
              type="text"
              id="username"
              class="form-control cozy-input"
              bind:value={username}
              required
            />
          </div>

          <div class="mb-4">
            <label for="password" class="cozy-label">Password</label>
            <input
              type="password"
              id="password"
              class="form-control cozy-input"
              bind:value={password}
              required
            />
          </div>

          <div class="text-center">
            <button
              type="submit"
              class="btn btn-success px-5 btn-lg rounded-0 w-100"
              disabled={loading}
            >
              {#if loading}
                <div
                  class="spinner-border spinner-border-sm me-2"
                  role="status"
                >
                  <span class="visually-hidden">Loading...</span>
                </div>
              {/if}
              Sign In
            </button>
          </div>
        </form>
      </div>
    </div>

    <style lang="scss">
    </style>

    <div class="mt-4 text-center">
      <p class="text-muted small">
        Don't have an account? <a href="/register">Sign up here</a>.
      </p>
      <p class="text-muted small">
        Access your personalized learning materials and study aids.
      </p>
    </div>
  </div>
</div>
