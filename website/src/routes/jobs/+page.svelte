<script lang="ts">
    import { onMount, onDestroy } from 'svelte';
    import { api } from '$lib/api/client';
    import { formatActivityType } from '$lib/utils';
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

<div class="well bg-white p-0">
    <table class="table table-hover mb-0">
        <thead>
            <tr>
                <th class="ps-3">Type</th>
                <th>Status</th>
                <th>Progress</th>
                <th>Message</th>
                <th class="pe-3">Started</th>
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
                        <td class="ps-3 small fw-bold">
                            {formatActivityType(job.type)}
                        </td>
                        <td>
                            <span class="d-flex align-items-center gap-1">
                                {#if job.status === 'COMPLETED'}
                                    <span class="glyphicon text-success"><CheckCircle2 size={14} /></span>
                                {:else if job.status === 'RUNNING'}
                                    <span class="glyphicon text-primary spin"><Loader2 size={14} /></span>
                                {:else if job.status === 'FAILED'}
                                    <span class="glyphicon text-danger"><XCircle size={14} /></span>
                                {:else if job.status === 'PENDING'}
                                    <span class="glyphicon text-secondary"><Play size={14} /></span>
                                {:else}
                                    <span class="glyphicon"><AlertCircle size={14} /></span>
                                {/if}
                                <span class="small">
                                    {#if job.status === 'COMPLETED'}
                                        Completed
                                    {:else if job.status === 'RUNNING'}
                                        Running
                                    {:else if job.status === 'FAILED'}
                                        Failed
                                    {:else if job.status === 'PENDING'}
                                        Queued
                                    {:else}
                                        {job.status}
                                    {/if}
                                </span>
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
                        <td class="pe-3 small text-muted">
                            {new Date(job.created_at).toLocaleString()}
                        </td>
                    </tr>
                {/each}
            {/if}
        </tbody>
    </table>
</div>

<style>
    .spin {
        animation: spin 2s linear infinite;
    }

    @keyframes spin {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }

    table thead th {
        border-top: none;
        font-weight: bold;
        font-size: 0.75rem;
        letter-spacing: 0.05em;
        color: #666;
        padding-top: 1rem;
        padding-bottom: 1rem;
    }

    .table-hover tbody tr:hover {
        background-color: #f9f9f9;
    }
</style>