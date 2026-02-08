<script lang="ts">
	import { notifications } from '$lib/notifications';
	import { X, Info, CheckCircle2, AlertCircle, AlertTriangle } from 'lucide-svelte';
	import { flip } from 'svelte/animate';
	import { fly } from 'svelte/transition';

	const icons = {
		info: Info,
		success: CheckCircle2,
		error: AlertCircle,
		warning: AlertTriangle
	};
</script>

<div class="notification-container">
	{#each $notifications as notification (notification.id)}
		<div 
			class="notification {notification.type}" 
			animate:flip={{ duration: 300 }}
			in:fly={{ y: 20, opacity: 0, duration: 300 }}
			out:fly={{ x: 100, opacity: 0, duration: 200 }}
		>
			<div class="icon-wrapper">
				<svelte:component this={icons[notification.type]} size={18} />
			</div>
			<div class="message">
				{notification.message}
			</div>
			<button class="close-btn" onclick={() => notifications.remove(notification.id)}>
				<X size={14} />
			</button>
		</div>
	{/each}
</div>

<style>
	.notification-container {
		position: fixed;
		bottom: var(--space-lg);
		right: var(--space-lg);
		display: flex;
		flex-direction: column;
		gap: var(--space-sm);
		z-index: 9999;
		pointer-events: none;
		max-width: 400px;
		width: calc(100% - 2 * var(--space-lg));
	}

	.notification {
		pointer-events: auto;
		background: #fff;
		border: 1px solid var(--border-color);
		border-radius: 4px;
		padding: var(--space-md);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
		display: flex;
		align-items: flex-start;
		gap: var(--space-sm);
		position: relative;
	}

	.icon-wrapper {
		flex-shrink: 0;
		padding-top: 2px;
	}

	.info .icon-wrapper { color: var(--accent-color); }
	.success .icon-wrapper { color: #226622; }
	.error .icon-wrapper { color: var(--error-color); }
	.warning .icon-wrapper { color: #856404; }

	.message {
		font-size: 13px;
		line-height: 1.4;
		color: #333;
		padding-right: var(--space-lg);
		word-break: break-word;
	}

	.close-btn {
		position: absolute;
		top: var(--space-sm);
		right: var(--space-sm);
		background: transparent;
		border: none;
		padding: 4px;
		cursor: pointer;
		color: #999;
		height: auto;
		min-width: auto;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.close-btn:hover {
		color: #333;
	}
</style>
