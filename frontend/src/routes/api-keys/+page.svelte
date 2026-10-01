<script lang="ts">
import { onMount } from 'svelte';
import { API_BASE, type APIKey, api } from '$api/client';
import { browser } from '$app/environment';
import { locale, t } from '$i18n';
import { addFlash } from '$stores/flash';

// Keep in sync with maxAPIKeysPerUser / allowedAPIKeyExpiryDays in backend handler/api_key.go.
const MAX_API_KEYS = 3;
const EXPIRY_OPTIONS = [30, 90, 180, 365];

let apiKeys = $state<APIKey[]>([]);
let newName = $state('');
let expiresInDays = $state(90);
let isLoading = $state(false);
let isCreating = $state(false);
let createdToken = $state<string | null>(null);
let copied = $state(false);

let limitReached = $derived(apiKeys.length >= MAX_API_KEYS);
let apiBaseURL = $derived(browser ? new URL(API_BASE, window.location.origin).href : API_BASE);
let usageExample = $derived(
	`curl -H "Authorization: Bearer <token>" "${apiBaseURL.replace(/\/$/, '')}/metrics/dora?repository=<id>&start=YYYY-MM-DD&end=YYYY-MM-DD"`,
);

onMount(loadAPIKeys);

async function loadAPIKeys() {
	isLoading = true;
	try {
		apiKeys = (await api.apiKeys.list()) ?? [];
	} catch (error) {
		console.error('Failed to load API keys:', error);
	}
	isLoading = false;
}

async function createAPIKey() {
	if (!newName.trim() || limitReached) return;
	isCreating = true;
	try {
		const res = await api.apiKeys.create(newName.trim(), expiresInDays);
		createdToken = res.token;
		copied = false;
		newName = '';
		await loadAPIKeys();
	} catch (error) {
		console.error('Failed to create API key:', error);
	}
	isCreating = false;
}

async function revokeAPIKey(key: APIKey) {
	if (!confirm($t('apiKeys.confirmRevoke', { name: key.name }))) return;
	try {
		await api.apiKeys.revoke(key.id);
		await loadAPIKeys();
		addFlash('success', $t('apiKeys.revokeSuccess', { name: key.name }));
	} catch (error) {
		console.error('Failed to revoke API key:', error);
		addFlash('error', $t('apiKeys.revokeFailed', { name: key.name }));
	}
}

async function copyToken() {
	if (!createdToken) return;
	try {
		await navigator.clipboard.writeText(createdToken);
		copied = true;
	} catch (error) {
		console.error('Failed to copy API key:', error);
	}
}

function closeCreated() {
	createdToken = null;
	copied = false;
}

function isExpired(key: APIKey): boolean {
	return key.expiresAt != null && new Date(key.expiresAt).getTime() <= Date.now();
}

function formatDate(value?: string): string {
	return value ? new Date(value).toLocaleDateString($locale) : '-';
}

function handleKeyDown(e: KeyboardEvent) {
	if (e.key === 'Enter') createAPIKey();
}
</script>

<svelte:head>
	<title>{$t('apiKeys.pageTitle')}</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<h1>🔑 {$t('apiKeys.title')}</h1>
	</header>

	<section class="section">
		<p class="section-desc">{$t('apiKeys.description')}</p>

		<div class="add-form">
			<input
				type="text"
				bind:value={newName}
				placeholder={$t('apiKeys.namePlaceholder')}
				class="input"
				maxlength="64"
				onkeydown={handleKeyDown}
				disabled={isCreating || limitReached}
				aria-label={$t('apiKeys.name')}
			>
			<select
				class="select"
				bind:value={expiresInDays}
				disabled={isCreating || limitReached}
				aria-label={$t('apiKeys.expiresIn')}
			>
				{#each EXPIRY_OPTIONS as days}
					<option value={days}>{$t('apiKeys.days', { days })}</option>
				{/each}
			</select>
			<button
				class="btn btn-primary"
				onclick={createAPIKey}
				disabled={isCreating || limitReached || !newName.trim()}
			>
				{isCreating ? $t('apiKeys.creating') : $t('apiKeys.create')}
			</button>
		</div>
		{#if limitReached}
			<p class="limit-note">{$t('apiKeys.limitReached', { max: MAX_API_KEYS })}</p>
		{/if}

		{#if createdToken}
			<div class="created-panel">
				<h2>{$t('apiKeys.createdTitle')}</h2>
				<p class="created-warning">{$t('apiKeys.createdWarning')}</p>
				<div class="token-row">
					<code class="token">{createdToken}</code>
					<button class="btn btn-primary btn-sm" onclick={copyToken}>
						{copied ? $t('apiKeys.copied') : $t('apiKeys.copy')}
					</button>
					<button class="btn btn-secondary btn-sm" onclick={closeCreated}>
						{$t('apiKeys.close')}
					</button>
				</div>
			</div>
		{/if}

		{#if isLoading}
			<div class="loading">{$t('common.loading')}</div>
		{:else if apiKeys.length === 0}
			<div class="empty-state">
				<p>{$t('apiKeys.noKeys')}</p>
			</div>
		{:else}
			<div class="card">
				<table class="table">
					<thead>
						<tr>
							<th>{$t('apiKeys.name')}</th>
							<th>{$t('apiKeys.keyPrefix')}</th>
							<th>{$t('apiKeys.createdAt')}</th>
							<th>{$t('apiKeys.lastUsed')}</th>
							<th>{$t('apiKeys.expiresAt')}</th>
							<th></th>
						</tr>
					</thead>
					<tbody>
						{#each apiKeys as key (key.id)}
							<tr>
								<td>{key.name}</td>
								<td>
									<code class="prefix">{key.prefix}_…</code>
								</td>
								<td>{formatDate(key.createdAt)}</td>
								<td>{key.lastUsedAt ? formatDate(key.lastUsedAt) : $t('apiKeys.neverUsed')}</td>
								<td>
									{formatDate(key.expiresAt)}
									{#if isExpired(key)}
										<span class="badge-expired">{$t('apiKeys.expired')}</span>
									{/if}
								</td>
								<td class="actions">
									<button class="btn btn-danger btn-sm" onclick={() => revokeAPIKey(key)}>
										{$t('apiKeys.revoke')}
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</section>

	<section class="section">
		<h2>{$t('apiKeys.usageTitle')}</h2>
		<p class="section-desc">{$t('apiKeys.usageDesc')}</p>
		<pre class="usage"><code>{usageExample}</code></pre>
	</section>
</div>

<style>
.page {
	max-width: 1200px;
	margin: 0 auto;
}

.page-header {
	margin-bottom: 2rem;
}

.section {
	margin-bottom: 2rem;
}

.section h2 {
	font-size: 1.125rem;
	margin-bottom: 0.5rem;
}

.section-desc {
	font-size: 0.875rem;
	color: var(--color-text-muted);
	margin-bottom: 1rem;
}

.add-form {
	display: flex;
	gap: 0.75rem;
	margin-bottom: 1rem;
}

.input,
.select {
	padding: 0.625rem 0.875rem;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-bg-card);
	color: var(--color-text);
	font-size: 0.875rem;
}

.input {
	flex: 1;
}

.input:focus,
.select:focus {
	outline: none;
	border-color: var(--color-primary);
	box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
}

.btn {
	padding: 0.625rem 1rem;
	border: none;
	border-radius: var(--radius-md);
	font-size: 0.875rem;
	font-weight: 500;
	cursor: pointer;
	transition: all 0.15s ease;
	white-space: nowrap;
}

.btn:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.btn-primary {
	background: var(--color-primary);
	color: white;
}

.btn-primary:hover:not(:disabled) {
	opacity: 0.9;
}

.btn-secondary {
	background: var(--color-bg);
	color: var(--color-text);
	border: 1px solid var(--color-border);
}

.btn-danger {
	background: #ef4444;
	color: white;
}

.btn-danger:hover:not(:disabled) {
	background: #dc2626;
}

.btn-sm {
	padding: 0.375rem 0.75rem;
	font-size: 0.75rem;
}

.limit-note {
	font-size: 0.8125rem;
	color: #f59e0b;
	margin-bottom: 1rem;
}

.created-panel {
	margin-bottom: 1.5rem;
	padding: 1rem 1.25rem;
	background: var(--color-bg-card);
	border: 1px solid #22c55e;
	border-radius: var(--radius-lg);
}

.created-panel h2 {
	font-size: 1rem;
	margin-bottom: 0.25rem;
}

.created-warning {
	font-size: 0.8125rem;
	color: #f59e0b;
	margin-bottom: 0.75rem;
}

.token-row {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.token {
	flex: 1;
	padding: 0.5rem 0.75rem;
	background: var(--color-bg);
	border-radius: var(--radius-md);
	font-family: var(--font-mono);
	font-size: 0.8125rem;
	word-break: break-all;
}

.loading {
	text-align: center;
	padding: 2rem;
	color: var(--color-text-muted);
}

.empty-state {
	text-align: center;
	padding: 2rem;
	color: var(--color-text-muted);
	background: var(--color-bg-card);
	border-radius: var(--radius-lg);
	border: 1px solid var(--color-border);
}

.card {
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	padding: 1.25rem;
	overflow-x: auto;
}

.table {
	width: 100%;
	border-collapse: collapse;
}

.table th,
.table td {
	padding: 0.75rem 1rem;
	text-align: left;
	border-bottom: 1px solid var(--color-border);
	font-size: 0.875rem;
}

.table th {
	font-size: 0.75rem;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	color: var(--color-text-muted);
	font-weight: 600;
}

.table tbody tr:hover {
	background: var(--color-bg);
}

.prefix {
	font-family: var(--font-mono);
	font-size: 0.8125rem;
}

.badge-expired {
	display: inline-block;
	margin-left: 0.5rem;
	padding: 0.125rem 0.5rem;
	border-radius: 999px;
	background: rgba(239, 68, 68, 0.15);
	color: #ef4444;
	font-size: 0.6875rem;
	font-weight: 600;
}

.actions {
	text-align: right;
}

.usage {
	padding: 0.75rem 1rem;
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	font-family: var(--font-mono);
	font-size: 0.8125rem;
	overflow-x: auto;
	white-space: pre-wrap;
	word-break: break-all;
}

@media (max-width: 640px) {
	.add-form,
	.token-row {
		flex-direction: column;
		align-items: stretch;
	}
}
</style>
