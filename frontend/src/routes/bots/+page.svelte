<script lang="ts">
import { onMount } from 'svelte';
import { api, type BotUser, type CycleTimeMetrics, type ReviewMetrics } from '$api/client';
import { addFlash } from '$stores/flash';
import { browser } from '$app/environment';
import { t } from '$i18n';
import { dateRange, formatHours } from '$stores/metrics';
import { selectedRepositories } from '$stores/repositories';

let botUsers = $state<BotUser[]>([]);
let newBotUsername = $state('');
let isLoadingBots = $state(false);
let isLoadingMetrics = $state(false);
let isAdding = $state(false);

// Bot metrics
let cycleTime = $state<CycleTimeMetrics | null>(null);
let reviewMetrics = $state<ReviewMetrics | null>(null);
let botsLoaded = $state(false);

onMount(async () => {
	await loadBotUsers();
	botsLoaded = true;
});

async function loadBotUsers() {
	isLoadingBots = true;
	try {
		botUsers = (await api.botUsers.list()) ?? [];
	} catch (error) {
		console.error('Failed to load bot users:', error);
	}
	isLoadingBots = false;
}

async function addBotUser() {
	if (!newBotUsername.trim()) return;
	isAdding = true;
	try {
		await api.botUsers.add(newBotUsername.trim());
		newBotUsername = '';
		await loadBotUsers();
	} catch (error) {
		console.error('Failed to add bot user:', error);
	}
	isAdding = false;
}

async function deleteBotUser(username: string) {
	if (!confirm($t('bots.confirmDelete', { name: username }))) return;

	try {
		await api.botUsers.delete(username);
		await loadBotUsers();
		addFlash('success', $t('bots.deleteSuccess', { name: username }));
	} catch (error) {
		console.error('Failed to delete bot user:', error);
		addFlash('error', $t('bots.deleteFailed', { name: username }));
	}
}

async function loadBotMetrics() {
	isLoadingMetrics = true;
	const repos = $selectedRepositories.length > 0 ? $selectedRepositories : undefined;
	const botFilter = { botsOnly: true };

	try {
		[cycleTime, reviewMetrics] = await Promise.all([
			api.metrics.cycleTime(repos, $dateRange.start, $dateRange.end, false, botFilter),
			api.metrics.reviews(repos, $dateRange.start, $dateRange.end, false, botFilter),
		]);
	} catch (error) {
		console.error('Failed to load bot metrics:', error);
	}
	isLoadingMetrics = false;
}

$effect(() => {
	if (browser && botsLoaded && $selectedRepositories) {
		loadBotMetrics();
	}
});

function handleKeyDown(e: KeyboardEvent) {
	if (e.key === 'Enter') addBotUser();
}
</script>

<svelte:head>
	<title>{$t('bots.pageTitle')}</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<h1>🤖 {$t('bots.title')}</h1>
	</header>

	<!-- Custom Bot Management -->
	<section class="section">
		<h2>{$t('bots.customBotUsers')}</h2>
		<p class="section-desc">{@html $t('bots.customBotDesc')}</p>

		<div class="add-form">
			<input
				type="text"
				bind:value={newBotUsername}
				placeholder={$t('bots.usernamePlaceholder')}
				class="input"
				onkeydown={handleKeyDown}
				disabled={isAdding}
			>
			<button
				class="btn btn-primary"
				onclick={addBotUser}
				disabled={isAdding || !newBotUsername.trim()}
			>
				{isAdding ? $t('common.adding') : $t('common.add')}
			</button>
		</div>

		{#if isLoadingBots}
			<div class="loading">{$t('common.loading')}</div>
		{:else if botUsers.length === 0}
			<div class="empty-state">
				<p>{$t('bots.noBots')}</p>
			</div>
		{:else}
			<div class="bot-list">
				{#each botUsers as bot}
					<div class="bot-item">
						<span class="bot-username">@{bot.username}</span>
						<span class="bot-date">{new Date(bot.createdAt).toLocaleDateString('ja-JP')}</span>
						<button class="btn btn-danger btn-sm" onclick={() => deleteBotUser(bot.username)}>
							{$t('common.delete')}
						</button>
					</div>
				{/each}
			</div>
		{/if}
	</section>

	<!-- Bot Metrics -->
	<section class="section">
		<h2>{$t('bots.botMetrics')}</h2>
		<p class="section-desc">{@html $t('bots.botMetricsDesc')}</p>

		{#if isLoadingMetrics}
			<div class="loading">{$t('common.loadingMetrics')}</div>
		{:else}
			<div class="summary-grid">
				<div class="summary-card">
					<h3>{$t('bots.botPRs')}</h3>
					<div class="summary-value">{cycleTime?.totalPRs ?? 0}</div>
				</div>
				<div class="summary-card">
					<h3>{$t('bots.avgCycleTime')}</h3>
					<div class="summary-value">
						{cycleTime?.avgCycleTime ? formatHours(cycleTime.avgCycleTime) : '-'}
					</div>
				</div>
				<div class="summary-card">
					<h3>{$t('bots.botReviews')}</h3>
					<div class="summary-value">{reviewMetrics?.totalReviews ?? 0}</div>
				</div>
				<div class="summary-card">
					<h3>{$t('bots.botComments')}</h3>
					<div class="summary-value">{reviewMetrics?.totalComments ?? 0}</div>
				</div>
			</div>

			<!-- PR Stats by Bot -->
			{#if cycleTime?.byAuthor && cycleTime.byAuthor.length > 0}
				<div class="card mt-2">
					<h3 class="card-title">{$t('bots.botPRStats')}</h3>
					<table class="table">
						<thead>
							<tr>
								<th>{$t('bots.bot')}</th>
								<th class="num">{$t('bots.prCount')}</th>
								<th class="num">{$t('bots.avgCT')}</th>
								<th class="num">{$t('bots.addedLines')}</th>
								<th class="num">{$t('bots.deletedLines')}</th>
							</tr>
						</thead>
						<tbody>
							{#each cycleTime.byAuthor as author}
								<tr>
									<td>@{author.author}</td>
									<td class="num">{author.prCount}</td>
									<td class="num">{formatHours(author.avgCycleTime)}</td>
									<td class="num additions">+{author.additions.toLocaleString()}</td>
									<td class="num deletions">-{author.deletions.toLocaleString()}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}

			<!-- Review Stats by Bot -->
			{#if reviewMetrics?.byReviewer && reviewMetrics.byReviewer.length > 0}
				<div class="card mt-2">
					<h3 class="card-title">{$t('bots.botReviewStats')}</h3>
					<table class="table">
						<thead>
							<tr>
								<th>{$t('bots.bot')}</th>
								<th class="num">{$t('bots.reviewCount')}</th>
								<th class="num">{$t('bots.commentCount')}</th>
								<th class="num">{$t('bots.approvalRate')}</th>
							</tr>
						</thead>
						<tbody>
							{#each reviewMetrics.byReviewer as reviewer}
								<tr>
									<td>@{reviewer.reviewer}</td>
									<td class="num">{reviewer.reviewCount}</td>
									<td class="num">{reviewer.commentCount}</td>
									<td class="num">{reviewer.approvalRate.toFixed(1)}%</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		{/if}
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

.section-desc :global(code) {
	padding: 0.125rem 0.375rem;
	background: var(--color-bg);
	border-radius: 4px;
	font-size: 0.8125rem;
}

.add-form {
	display: flex;
	gap: 0.75rem;
	margin-bottom: 1.5rem;
}

.input {
	flex: 1;
	padding: 0.625rem 0.875rem;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	background: var(--color-bg-card);
	color: var(--color-text);
	font-size: 0.875rem;
}

.input:focus {
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

.bot-list {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.bot-item {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 0.75rem 1rem;
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.bot-username {
	font-weight: 500;
	font-family: var(--font-mono);
	flex: 1;
}

.bot-date {
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.summary-grid {
	display: grid;
	grid-template-columns: repeat(4, 1fr);
	gap: 1rem;
}

.summary-card {
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	padding: 1.25rem;
	text-align: center;
}

.summary-card h3 {
	font-size: 0.75rem;
	color: var(--color-text-muted);
	text-transform: uppercase;
	margin-bottom: 0.5rem;
}

.summary-value {
	font-size: 1.5rem;
	font-weight: 700;
	color: var(--color-primary);
}

.card {
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	padding: 1.25rem;
}

.card-title {
	font-size: 0.875rem;
	font-weight: 600;
	margin-bottom: 1rem;
}

.mt-2 {
	margin-top: 1.5rem;
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

.num {
	text-align: right;
	font-variant-numeric: tabular-nums;
}

.additions {
	color: #22c55e;
}

.deletions {
	color: #ef4444;
}

@media (max-width: 1024px) {
	.summary-grid {
		grid-template-columns: repeat(2, 1fr);
	}
}

@media (max-width: 640px) {
	.summary-grid {
		grid-template-columns: 1fr;
	}

	.add-form {
		flex-direction: column;
	}
}
</style>
