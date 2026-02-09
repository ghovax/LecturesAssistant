<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { api } from '$lib/api/client';
    import Breadcrumb from '$lib/components/Breadcrumb.svelte';
    import { Loader2, CheckCircle2, XCircle, Play, AlertCircle } from 'lucide-svelte';

    let jobs = $state<any[]>([]);
    let loading = $state(true);
    let interval: any;

    async function loadJobs() {
        try {
            const data = await api.listJobs();
            jobs = data ?? [];
        } catch (e) {
            console.error(e);
        } finally {
            loading = false;
        }
    }

    onMount(() => {
        loadJobs();
        interval = setInterval(loadJobs, 3000);
    });

    onDestroy(() => clearInterval(interval));
</script>

<Breadcrumb items={[{ label: 'Activity Progress', active: true }]} />

<h2>Activity Progress</h2>

<p class="text-muted">Keep an eye on how your transcriptions and study guides are progressing.</p>

<div class="well bg-white p-0 overflow-hidden shadow-sm">
    <table class="table table-hover mb-0">
        <thead class="bg-light">
            <tr>
                <th>Type</th>
                <th>Status</th>
                <th>Progress</th>
                <th>Message</th>
                <th>Started</th>
            </tr>
        </thead>
        <tbody>
            {#if loading && jobs.length === 0}
                <tr><td colspan="5" class="text-center p-5"><div class="village-spinner mx-auto"></div></td></tr>
            {:else if jobs.length === 0}
                <tr><td colspan="5" class="text-center p-5">No recent jobs found.</td></tr>
            {:else}
                {#each jobs as job}
                    <tr>
                        <td class="small fw-bold text-uppercase">{job.type.replace('_', ' ')}</td>
                        <td>
                            <span class="d-flex align-items-center gap-1">
                                {#if job.status === 'COMPLETED'}
                                    <CheckCircle2 size={14} class="text-success" />
                                {:else if job.status === 'RUNNING'}
                                    <Loader2 size={14} class="text-primary spin" />
                                {:else if job.status === 'FAILED'}
                                    <XCircle size={14} class="text-danger" />
                                {:else if job.status === 'PENDING'}
                                    <Play size={14} class="text-secondary" />
                                {:else}
                                    <AlertCircle size={14} />
                                {/if}
                                <span class="small">{job.status}</span>
                            </span>
                        </td>
                        <td style="width: 150px;">
                            <div class="progress" style="height: 6px; margin-top: 8px;">
                                <div 
                                    class="progress-bar {job.status === 'COMPLETED' ? 'bg-success' : (job.status === 'FAILED' ? 'bg-danger' : 'bg-primary')}" 
                                    style="width: {job.progress}%"
                                ></div>
                            </div>
                        </td>
                        <td class="small text-muted text-truncate" style="max-width: 250px;">
                            {job.progress_message_text || '-'}
                        </td>
                        <td class="small text-muted">
                            {new Date(job.created_at).toLocaleString()}
                        </td>
                    </tr>
                {/each}
            {/if}
        </tbody>
    </table>
</div>
