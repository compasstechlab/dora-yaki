<script lang="ts">
import { fly } from 'svelte/transition';
import { type FlashMessage, flashMessages, removeFlash } from '$stores/flash';

const iconMap: Record<FlashMessage['type'], string> = {
	error: '✕',
	warning: '⚠',
	success: '✓',
	info: 'ℹ',
};
</script>

{#if $flashMessages.length > 0}
	<div class="flash-container">
		{#each $flashMessages as msg (msg.id)}
			<div class="flash flash-{msg.type}" transition:fly={{ y: -20, duration: 250 }}>
				<span class="flash-icon">{iconMap[msg.type]}</span>
				<span class="flash-text">{msg.message}</span>
				<button class="flash-close" onclick={() => removeFlash(msg.id)}>×</button>
			</div>
		{/each}
	</div>
{/if}

<style>
.flash-container {
	position: fixed;
	top: 1rem;
	left: 50%;
	transform: translateX(-50%);
	z-index: 9999;
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
	max-width: 500px;
	width: 90%;
}

.flash {
	display: flex;
	align-items: center;
	gap: 0.625rem;
	padding: 0.75rem 1rem;
	border-radius: var(--radius-md, 8px);
	font-size: 0.875rem;
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
	animation: none;
}

.flash-error {
	background: #fef2f2;
	color: #991b1b;
	border: 1px solid #fecaca;
}

.flash-warning {
	background: #fffbeb;
	color: #92400e;
	border: 1px solid #fde68a;
}

.flash-success {
	background: #f0fdf4;
	color: #166534;
	border: 1px solid #bbf7d0;
}

.flash-info {
	background: #eff6ff;
	color: #1e40af;
	border: 1px solid #bfdbfe;
}

.flash-icon {
	flex-shrink: 0;
	font-weight: 700;
	font-size: 1rem;
}

.flash-text {
	flex: 1;
	line-height: 1.4;
}

.flash-close {
	flex-shrink: 0;
	background: none;
	border: none;
	font-size: 1.125rem;
	cursor: pointer;
	color: inherit;
	opacity: 0.6;
	padding: 0;
	line-height: 1;
}

.flash-close:hover {
	opacity: 1;
}
</style>
