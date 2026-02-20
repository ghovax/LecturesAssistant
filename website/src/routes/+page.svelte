<script lang="ts">
  import { onMount } from "svelte";
  import { auth } from "$lib/auth.svelte";
  import { api } from "$lib/api/client";
  import { notifications } from "$lib/stores/notifications.svelte";
  import { goto } from "$app/navigation";
  import Tile from "$lib/components/Tile.svelte";
  import {
    Book,
    Settings,
    LogIn,
    LogOut,
    User,
    HelpCircle,
    Heart,
    Database,
    Download,
    Upload,
  } from "lucide-svelte";
  import { ConfirmModal } from "$lib";

  async function handleLogout() {
    await auth.logout();
    goto("/");
  }

  // Confirmation Modal State
  let confirmModal = $state({
    isOpen: false,
    title: "",
    message: "",
    confirmText: "Confirm",
    onConfirm: () => {},
    isDanger: false,
  });

  function showConfirm(options: {
    title: string;
    message: string;
    confirmText?: string;
    onConfirm: () => void;
    isDanger?: boolean;
  }) {
    confirmModal = {
      isOpen: true,
      title: options.title,
      message: options.message,
      confirmText: options.confirmText ?? "Confirm",
      onConfirm: () => {
        options.onConfirm();
        confirmModal.isOpen = false;
      },
      isDanger: options.isDanger ?? false,
    };
  }

  async function handleBackup() {
    try {
      const token = localStorage.getItem("session_token");
      const url = api.getBaseUrl() + "/system/backup?session_token=" + token;

      // Use a hidden anchor to trigger download
      const link = document.createElement("a");
      link.href = url;
      link.setAttribute("download", "");
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);

      notifications.success("Starting database backup download...");
    } catch (e: any) {
      notifications.error("Failed to start backup: " + (e.message || e));
    }
  }

  let restoreFileInput: HTMLInputElement | null = $state(null);
  let selectedRestoreFile: File | null = $state(null);

  function triggerRestore() {
    selectedRestoreFile = null;
    showConfirm({
      title: "Restore Workspace",
      message:
        "This is a destructive action. Restoring a backup will permanently overwrite all your current subjects, lessons, and configurations.",
      confirmText: "Select File",
      isDanger: true,
      onConfirm: () => {
        restoreFileInput?.click();
      },
    });
  }

  async function handleRestoreFile(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    selectedRestoreFile = file;

    // Update modal to show confirmation with file selected
    showConfirm({
      title: "Restore Workspace",
      message: `This will permanently overwrite all your current subjects, lessons, and configurations. Are you sure?`,
      confirmText: "Yes, Replace Everything",
      isDanger: true,
      onConfirm: async () => {
        try {
          notifications.info("Restoring workspace, please wait...");
          await api.restoreDatabase(file);
          notifications.success(
            "Workspace restored successfully. Refreshing...",
          );
          setTimeout(() => window.location.reload(), 1500);
        } catch (e: any) {
          notifications.error(
            "Failed to restore workspace: " + (e.message || e),
          );
        } finally {
          input.value = "";
          selectedRestoreFile = null;
        }
      },
    });
  }

  onMount(async () => {
    // Scroll Motion Blur Observer
    const scrollObserver = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add("visible");
          }
        });
      },
      {
        threshold: 0.05,
        rootMargin: "0px 0px -20px 0px",
      },
    );

    document
      .querySelectorAll(".scroll-blur, .scroll-blur-heavy, .scroll-blur-light")
      .forEach((el) => {
        scrollObserver.observe(el);
      });

    await auth.check();
  });
</script>

<div class="cozy-homepage">
  <ConfirmModal
    isOpen={confirmModal.isOpen}
    title={confirmModal.title}
    message={confirmModal.message}
    confirmText={confirmModal.confirmText}
    isDanger={confirmModal.isDanger}
    onConfirm={confirmModal.onConfirm}
    onCancel={() => (confirmModal.isOpen = false)}
  />

  <input
    type="file"
    accept=".db"
    class="d-none"
    bind:this={restoreFileInput}
    onchange={handleRestoreFile}
  />

  <header class="hero-section">
    <span class="overline scroll-blur-light visible">Welcome to your</span>
    <h1 class="scroll-blur-heavy visible">
      Lectures<span class="text-orange">Assistant</span>
    </h1>
    <p class="subtitle scroll-blur-light visible">
      A minimalist, cozy space to organize your studies, transcribe recordings,
      and generate smart materials for your exams.
    </p>
    <div class="mt-4 scroll-blur-light visible d-flex align-items-center gap-3">
      <a href="/exams" class="btn btn-primary btn-lg px-5">
        Start My Studies
      </a>
      <a
        href="https://ko-fi.com/giovannigravili59139"
        target="_blank"
        rel="noopener noreferrer"
        class="donate-btn-hero"
        title="Support the project on Ko-fi"
      >
        <img src="/logomarkLogo.webp" alt="Ko-fi" class="kofii-icon" />
        <span class="donate-text">Buy me a coffee</span>
      </a>
    </div>
  </header>

  <section class="scroll-blur">
    <div class="section-header">
      <span class="overline">Workspace</span>
      <h2>Core Study Tools</h2>
    </div>

    <div class="link-tiles mb-4">
      <Tile href="/exams" icon="" title="My Studies">
        {#snippet description()}
          Access subjects, lessons, and all generated materials.
        {/snippet}
      </Tile>
      <Tile href="/settings" icon="" title="Preferences">
        {#snippet description()}
          Customize language, AI models, and interface.
        {/snippet}
      </Tile>
    </div>
  </section>

  <div class="row g-4 scroll-blur mb-5">
    <div class="col-lg-6">
      <section class="h-100 mb-0">
        <div class="section-header">
          <span class="overline">Identity</span>
          <h2>Account & Session</h2>
        </div>
        <div class="link-tiles">
          {#if !auth.user}
            <Tile href="/login" icon="" title="Sign In">
              {#snippet description()}
                Access your personal study hub.
              {/snippet}
            </Tile>
          {:else}
            <Tile onclick={handleLogout} icon="" title="Logout">
              {#snippet description()}
                Securely sign out of your session.
              {/snippet}
            </Tile>
            {#if auth.user?.role === "admin"}
              <Tile
                icon=""
                title="Workspace Backup"
                class="restore-workspace-tile"
              >
                {#snippet description()}
                  Export or restore your study library.
                {/snippet}
                {#snippet actions()}
                  <button
                    type="button"
                    class="btn btn-link p-0 border-0 shadow-none"
                    onclick={handleBackup}
                    title="Export Workspace"
                  >
                    <Download size={18} />
                  </button>
                  <button
                    type="button"
                    class="btn btn-link text-danger p-0 border-0 shadow-none"
                    onclick={triggerRestore}
                    title="Restore Workspace"
                  >
                    <Upload size={18} />
                  </button>
                {/snippet}
              </Tile>
            {/if}
          {/if}
        </div>
      </section>
    </div>
    <div class="col-lg-6">
      <section class="h-100 mb-0">
        <div class="section-header">
          <span class="overline">Support</span>
          <h2>Resources</h2>
        </div>
        <div class="link-tiles">
          <Tile href="/help" icon="" title="Help Guide">
            {#snippet description()}
              How to use the assistant effectively.
            {/snippet}
          </Tile>
          <Tile href="/credits" icon="" title="Credits">
            {#snippet description()}
              System acknowledgments and contributors.
            {/snippet}
          </Tile>
        </div>
      </section>
    </div>
  </div>

  <footer class="cozy-footer scroll-blur-light">
    <p>Built with craft principles for a focused learning experience.</p>
  </footer>
</div>

<style lang="scss">
  .cozy-homepage {
    font-family: var(--font-primary);
    color: var(--gray-800);
    max-width: 1300px;
    margin: 0 auto;
    padding-bottom: 80px;
    -webkit-font-smoothing: antialiased;
  }

  .hero-section {
    padding: 80px 0 60px;
    text-align: left;
  }

  .overline {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.12em;
    color: var(--gray-500);
    margin-bottom: 12px;
    display: block;
  }

  h1 {
    font-size: 2.25rem;
    font-weight: 500;
    margin-bottom: 20px;
    color: var(--gray-900);
    line-height: 1.1;
    letter-spacing: -0.02em;

    .text-orange {
      color: var(--orange);
      font-weight: 600;
    }
  }

  .subtitle {
    font-size: 1rem;
    font-weight: 400;
    color: var(--gray-600);
    line-height: 1.6;
    max-width: 560px;
  }

  section {
    margin-bottom: 60px;
  }

  .section-header {
    margin-bottom: 24px;

    h2 {
      font-size: 18px;
      font-weight: 500;
      color: var(--gray-900);
      margin: 0;
    }
  }

  .donate-btn-hero {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    text-decoration: none;
    color: var(--gray-700);
    font-size: 1rem;
    font-weight: 500;
    padding: 0.75rem 1.5rem;
    border: 1px solid var(--gray-300);
    border-radius: var(--border-radius);
    transition:
      background-color 0.2s ease,
      border-color 0.2s ease,
      color 0.2s ease;
    background: #fff;
    height: 48px;

    &:hover {
      background: #fff;
      border-color: var(--gray-400);
      color: var(--gray-900);
      text-decoration: none;
    }

    .kofii-icon {
      width: 24px;
      height: auto;
      flex-shrink: 0;
    }

    .donate-text {
      font-size: 0.95rem;
      white-space: nowrap;
    }
  }

  .link-tiles {
    display: flex;
    flex-wrap: wrap;
    gap: 0;
    background: transparent;
    margin-bottom: 2rem;
    border: 1px solid var(--gray-300);
    border-radius: var(--border-radius);
    overflow: visible;

    :global(.action-tile),
    :global(.tile-wrapper) {
      width: 250px;
      border-right: 1px solid var(--gray-300);
      border-radius: 0;

      &:last-child {
        border-right: none;
        border-radius: 0 var(--border-radius) var(--border-radius) 0;
      }

      &:first-child {
        border-radius: var(--border-radius) 0 0 var(--border-radius);
      }
    }

    .restore-workspace-tile {
      :global(.action-tile) {
        border: 2px solid #b91c1c;
      }
    }
  }

  .cozy-footer {
    padding-top: 40px;
    border-top: 1px solid var(--gray-300);
    text-align: center;
    margin-top: 40px;
    p {
      font-size: 12px;
      color: var(--gray-400);
    }
  }

  /* Scroll Motion Blur */
  .scroll-blur,
  .scroll-blur-heavy,
  .scroll-blur-light {
    opacity: 0;
    transform: translateY(20px);
    filter: blur(10px);
    transition: all 0.8s cubic-bezier(0.16, 1, 0.3, 1);

    &.visible {
      opacity: 1;
      transform: translateY(0);
      filter: blur(0);
    }
  }

  .scroll-blur-heavy {
    transition-duration: 1.2s;
    filter: blur(15px);
  }

  .scroll-blur-light {
    transition-duration: 0.6s;
    filter: blur(5px);
  }

  .donate-tile {
    :global(.action-tile) {
      border: 2px solid #ff5e5b;
    }

    .kofii-icon {
      width: 20px;
      height: 20px;
      position: absolute;
      top: 12px;
      right: 12px;
    }
  }

  @media (max-width: 768px) {
    .hero-section {
      padding: 60px 0 40px;
    }

    h1 {
      font-size: 32px;
    }

    .cozy-homepage {
      padding-left: 20px;
      padding-right: 20px;
    }
  }
</style>
