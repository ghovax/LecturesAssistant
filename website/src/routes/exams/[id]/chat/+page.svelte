<script lang="ts">
    import { onMount } from 'svelte';
    import { page } from '$app/state';
    import { api } from '$lib/api/client';
    import { goto } from '$app/navigation';

    let examId = $derived(page.params.id);

    onMount(async () => {
        try {
            const sessions = await api.request('GET', `/chat/sessions?exam_id=${examId}`);
            
            if (sessions && sessions.length > 0) {
                // Redirect to the most recent session
                goto(`/exams/${examId}/chat/${sessions[0].id}`, { replaceState: true });
            } else {
                // Create a new default session
                const session = await api.request('POST', '/chat/sessions', { 
                    exam_id: examId, 
                    title: 'Study Session 1' 
                });
                goto(`/exams/${examId}/chat/${session.id}`, { replaceState: true });
            }
        } catch (e) {
            console.error(e);
            goto(`/exams/${examId}`);
        }
    });
</script>

<div class="text-center p-5">
    <div class="village-spinner mx-auto mb-3"></div>
    <p class="text-muted">Entering Study Assistant...</p>
</div>
