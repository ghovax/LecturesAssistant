<script lang="ts">
  import { api } from "$lib/api/client";
  import { notifications } from "$lib/stores/notifications.svelte";
  import { goto } from "$app/navigation";
  import Breadcrumb from "$lib/components/Breadcrumb.svelte";

  let username = $state("");
  let password = $state("");
  let confirmPassword = $state("");
  let error = $state("");
  let loading = $state(false);

  async function handleRegister() {
    if (password !== confirmPassword) {
      error = "Passwords do not match";
      return;
    }

    loading = true;
    error = "";
    try {
      await api.request("POST", "/auth/register", { username, password });
      notifications.success(
        "Your account has been created. You can now sign in.",
      );
      goto("/login");
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }
</script>

<Breadcrumb items={[{ label: "Sign Up", active: true }]} />

<div class="row justify-content-center">
  <div class="col-lg-6">
    <div class="bg-white border mb-3">
      <div class="standard-header">
        <div class="header-title">
          <span class="header-text">Sign Up</span>
        </div>
      </div>

      <div class="p-4">
        <form
          onsubmit={(e) => {
            e.preventDefault();
            handleRegister();
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
            <label for="username" class="cozy-label">Desired Username</label>
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
              minlength="8"
            />
            <small class="form-text text-muted mt-2 d-block"
              >Minimum 8 characters.</small
            >
          </div>

          <div class="mb-4">
            <label for="confirmPassword" class="cozy-label"
              >Confirm Password</label
            >
            <input
              type="password"
              id="confirmPassword"
              class="form-control cozy-input"
              bind:value={confirmPassword}
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
              Create Account
            </button>
          </div>
        </form>
      </div>
    </div>

    <style lang="scss">
    </style>

    <div class="mt-4 text-center">
      <p class="text-muted small">
        Already have an account? <a href="/login">Log in here</a>.
      </p>
    </div>
  </div>
</div>
