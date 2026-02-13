<script lang="ts">
    import { onMount } from 'svelte';
    import { auth } from '$lib/auth.svelte';
    import { goto } from '$app/navigation';
    import Tile from '$lib/components/Tile.svelte';
    import { Book, Settings, LogIn, LogOut, User, HelpCircle, Heart } from 'lucide-svelte';

    async function handleLogout() {
        await auth.logout();
        goto('/');
    }

    onMount(async () => {
        // Scroll Motion Blur Observer
        const scrollObserver = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.classList.add('visible');
                }
            });
        }, {
            threshold: 0.05,
            rootMargin: '0px 0px -20px 0px'
        });

        document.querySelectorAll('.scroll-blur, .scroll-blur-heavy, .scroll-blur-light').forEach(el => {
            scrollObserver.observe(el);
        });

        await auth.check();
    });
</script>

<div class="cozy-homepage">
    <header class="hero-section">
        <span class="overline scroll-blur-light visible">Welcome to your</span>
        <h1 class="scroll-blur-heavy visible">
            Lectures<span class="text-orange">Assistant</span>
        </h1>
        <p class="subtitle scroll-blur-light visible">
            A minimalist, cozy space to organize your studies, transcribe recordings, and generate smart materials for your exams.
        </p>
    </header>

    <section class="scroll-blur">
        <div class="section-header">
            <span class="overline">Workspace</span>
            <h2>Core Study Tools</h2>
        </div>
        
        <div class="linkTiles mb-4">
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
                <div class="linkTiles">
                    {#if !auth.user}
                        <Tile href="/login" icon="" title="Sign In">
                            {#snippet description()}
                                Access your personal study hub.
                            {/snippet}
                        </Tile>
                    {:else}
                        <Tile href="/profile" icon="" title="My Profile">
                            {#snippet description()}
                                View and manage account details.
                            {/snippet}
                        </Tile>
                        <Tile onclick={handleLogout} icon="" title="Logout">
                            {#snippet description()}
                                Securely sign out of your session.
                            {/snippet}
                        </Tile>
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
                <div class="linkTiles">
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
        font-family: 'Manrope', sans-serif;
        color: var(--gray-800);
        max-width: 800px;
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
        font-size: 42px;
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
        font-size: 17px;
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

    .linkTiles {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
        gap: 0;
        background: transparent;
        margin-bottom: 2rem;
        overflow: hidden;
        
        :global(.tile-wrapper) {
            width: 100%;
            
            :global(a), :global(button) {
                width: 100%;
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
    .scroll-blur, .scroll-blur-heavy, .scroll-blur-light {
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
