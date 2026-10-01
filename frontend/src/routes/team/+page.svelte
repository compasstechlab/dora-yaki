<script lang="ts">
import { api, type FileExtensionMetrics, type MemberStats } from '$api/client';
import { browser } from '$app/environment';
import { t } from '$i18n';
import { isCodeExtension, isConfigExtension } from '$lib/utils/extensions';
import { dateRange, formatHours } from '$stores/metrics';
import { selectedRepositories } from '$stores/repositories';

const SKELETON_CARD_COUNT = 6;

// null until the first response arrives; kept during reloads so the grid does not flash.
let memberStats = $state<MemberStats[] | null>(null);
let isLoading = $state(false);
let loadError = $state<string | null>(null);
let hideInactive = $state(true);
let codeExtOnly = $state(true);
let configExtOnly = $state(false);
// Ignores responses from superseded requests when filters change quickly.
let requestSeq = 0;

function isActiveMember(stats: MemberStats): boolean {
	return (
		stats.prsAuthored > 0 ||
		stats.prsMerged > 0 ||
		stats.reviewsGiven > 0 ||
		stats.totalAdditions > 0 ||
		stats.totalDeletions > 0
	);
}

let allStats = $derived(memberStats ?? []);

let visibleStats = $derived(allStats.filter((s) => !hideInactive || isActiveMember(s)));

// Aggregate file extension stats across all team members
let teamFileExtStats = $derived.by(() => {
	const map = new Map<string, FileExtensionMetrics>();
	for (const stats of allStats) {
		if (!stats.byFileExtension) continue;
		for (const ext of stats.byFileExtension) {
			const existing = map.get(ext.extension);
			if (existing) {
				existing.additions += ext.additions;
				existing.deletions += ext.deletions;
				existing.files += ext.files;
				existing.prCount += ext.prCount;
			} else {
				map.set(ext.extension, { ...ext });
			}
		}
	}
	return [...map.values()].sort((a, b) => b.additions + b.deletions - (a.additions + a.deletions));
});

async function loadAllMemberStats(repos: string[] | undefined, start: string, end: string) {
	const seq = ++requestSeq;
	isLoading = true;
	loadError = null;
	try {
		const stats = await api.team.getAllMemberStats(repos, start, end);
		if (seq === requestSeq) memberStats = stats;
	} catch (error) {
		console.error('Failed to load member stats:', error);
		if (seq === requestSeq) loadError = $t('memberDetail.loadError');
	} finally {
		if (seq === requestSeq) isLoading = false;
	}
}

$effect(() => {
	if (!browser) return;
	const repos = $selectedRepositories.length > 0 ? $selectedRepositories : undefined;
	loadAllMemberStats(repos, $dateRange.start, $dateRange.end);
});
</script>

<svelte:head>
	<title>{$t('team.pageTitle')}</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<h1>{$t('team.title')}</h1>
		{#if isLoading && memberStats !== null}
			<div class="loading-inline" role="status">
				<div class="spinner"></div>
				<span>{$t('common.loadingStats')}</span>
			</div>
		{/if}
		<label class="toggle-inactive">
			<input type="checkbox" bind:checked={hideInactive}>
			<span>{$t('common.activeOnly')}</span>
		</label>
	</header>

	{#if memberStats === null}
		{#if loadError}
			<div class="empty-state">
				<p>{loadError}</p>
			</div>
		{:else}
			<div class="team-grid" aria-busy="true" aria-label={$t('common.loadingStats')}>
				{#each Array.from({ length: SKELETON_CARD_COUNT }, (_, i) => i) as i (i)}
					<div class="member-card" aria-hidden="true">
						<div class="member-header">
							<div class="skeleton skeleton-avatar"></div>
							<div class="member-info">
								<div class="skeleton skeleton-line skeleton-line-name"></div>
								<div class="skeleton skeleton-line skeleton-line-login"></div>
							</div>
						</div>
						<div class="skeleton skeleton-stats"></div>
						<div class="skeleton skeleton-line skeleton-line-code"></div>
					</div>
				{/each}
			</div>
		{/if}
	{:else if allStats.length === 0}
		<div class="empty-state">
			<p>{$t('team.noMembers')}</p>
		</div>
	{:else}
		{#if loadError}
			<div class="error-banner" role="alert">{loadError}</div>
		{/if}
		{#if visibleStats.length === 0 && hideInactive}
			<div class="empty-state">
				<p>{$t('team.noActiveMembers')}</p>
			</div>
		{/if}
		<div class="team-grid" class:is-stale={isLoading} aria-busy={isLoading}>
			{#each visibleStats as stats (stats.member.id)}
				{@const member = stats.member}
				<a href="/team/{member.id}" class="member-card member-card-link">
					<div class="member-header">
						{#if member.avatarUrl}
							<img src={member.avatarUrl} alt={member.login} class="avatar">
						{:else}
							<div class="avatar-placeholder">{member.login[0].toUpperCase()}</div>
						{/if}
						<div class="member-info">
							<h3 class="member-name">{member.name || member.login}</h3>
							<span class="member-login">@{member.login}</span>
						</div>
					</div>

					<div class="member-stats">
						<div class="stat">
							<span class="stat-value">{stats.prsAuthored}</span>
							<span class="stat-label">{$t('team.prsCreated')}</span>
						</div>
						<div class="stat">
							<span class="stat-value">{stats.prsMerged}</span>
							<span class="stat-label">{$t('team.merged')}</span>
						</div>
						<div class="stat">
							<span class="stat-value">{stats.reviewsGiven}</span>
							<span class="stat-label">{$t('team.reviews')}</span>
						</div>
						<div class="stat">
							<span class="stat-value">
								{stats.avgCycleTime > 0 ? formatHours(stats.avgCycleTime) : "-"}
							</span>
							<span class="stat-label">{$t('team.avgCT')}</span>
						</div>
					</div>

					<div class="code-stats">
						<span class="additions">+{stats.totalAdditions.toLocaleString()}</span>
						<span class="deletions">-{stats.totalDeletions.toLocaleString()}</span>
					</div>

					{#if stats.byFileExtension && stats.byFileExtension.length > 0}
						<div class="ext-stats">
							{#each stats.byFileExtension.slice(0, 3) as ext}<span class="ext-tag">
								<code>{ext.extension}</code>
								<span class="ext-additions">+{ext.additions.toLocaleString()}</span>
								<span class="ext-deletions">-{ext.deletions.toLocaleString()}</span>
							</span>{/each}
						</div>
					{/if}
				</a>
			{/each}
		</div>

		<!-- Team Summary -->
		<section class="section">
			<h2>{$t('team.teamSummary')}</h2>
			<div class="summary-grid">
				<div class="summary-card">
					<h3>{$t('team.totalPRs')}</h3>
					<div class="summary-value">{allStats.reduce((sum, s) => sum + s.prsAuthored, 0)}</div>
				</div>
				<div class="summary-card">
					<h3>{$t('team.totalReviews')}</h3>
					<div class="summary-value">{allStats.reduce((sum, s) => sum + s.reviewsGiven, 0)}</div>
				</div>
				<div class="summary-card">
					<h3>{$t('team.activeMembers')}</h3>
					<div class="summary-value">
						{allStats.filter((s) => s.prsAuthored > 0 || s.reviewsGiven > 0).length}
					</div>
				</div>
				<div class="summary-card">
					<h3>{$t('team.totalCodeChanges')}</h3>
					<div class="summary-value">
						{allStats
							.reduce((sum, s) => sum + s.totalAdditions + s.totalDeletions, 0)
							.toLocaleString()}
						lines
					</div>
				</div>
			</div>
		</section>

		<!-- Team File Extension Stats -->
		{#if teamFileExtStats.length > 0}
			{@const filtered = teamFileExtStats.filter(e => {
        if (codeExtOnly && configExtOnly) return isCodeExtension(e.extension) || isConfigExtension(e.extension);
        if (codeExtOnly) return isCodeExtension(e.extension);
        if (configExtOnly) return isConfigExtension(e.extension);
        return true;
      })}
			{@const top10 = filtered.slice(0, 10)}
			{@const rest = filtered.slice(10)}
			{@const other =
        rest.length > 0
          ? {
              extension: $t('common.other'),
              additions: rest.reduce((s, r) => s + r.additions, 0),
              deletions: rest.reduce((s, r) => s + r.deletions, 0),
              files: rest.reduce((s, r) => s + r.files, 0),
              prCount: rest.reduce((s, r) => s + r.prCount, 0),
            }
          : null}
			<section class="section">
				<div class="section-header">
					<h2>{$t('team.teamFileExt')}</h2>
					<label class="toggle-code-ext">
						<input type="checkbox" bind:checked={codeExtOnly}>
						<span>{$t('common.codeFilesOnly')}</span>
					</label>
					<label class="toggle-code-ext">
						<input type="checkbox" bind:checked={configExtOnly}>
						<span>{$t('common.configFilesOnly')}</span>
					</label>
				</div>
				<div class="card">
					<table class="table">
						<thead>
							<tr>
								<th>{$t('team.extension')}</th>
								<th class="num">{$t('team.addedLines')}</th>
								<th class="num">{$t('team.deletedLines')}</th>
								<th class="num">{$t('team.fileCount')}</th>
								<th class="num">{$t('team.prCount')}</th>
							</tr>
						</thead>
						<tbody>
							{#each top10 as ext}
								<tr>
									<td>
										<code>{ext.extension}</code>
									</td>
									<td class="num ext-additions">+{ext.additions.toLocaleString()}</td>
									<td class="num ext-deletions">-{ext.deletions.toLocaleString()}</td>
									<td class="num">{ext.files.toLocaleString()}</td>
									<td class="num">{ext.prCount}</td>
								</tr>
							{/each}
							{#if other}
								<tr class="other-row">
									<td>{other.extension}</td>
									<td class="num ext-additions">+{other.additions.toLocaleString()}</td>
									<td class="num ext-deletions">-{other.deletions.toLocaleString()}</td>
									<td class="num">{other.files.toLocaleString()}</td>
									<td class="num">{other.prCount}</td>
								</tr>
							{/if}
						</tbody>
					</table>
				</div>
			</section>
		{/if}
	{/if}
</div>

<style>
.page {
	max-width: 1400px;
	margin: 0 auto;
}

.page-header {
	display: flex;
	align-items: center;
	gap: 1rem;
	margin-bottom: 2rem;
}

.toggle-inactive {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 0.8125rem;
	color: var(--color-text-muted);
	cursor: pointer;
	margin-left: auto;
	white-space: nowrap;
}

.toggle-inactive input[type="checkbox"] {
	accent-color: var(--color-primary);
	width: 14px;
	height: 14px;
}

.section-header {
	display: flex;
	align-items: center;
	gap: 1rem;
	margin-bottom: 1rem;
}

.section-header h2 {
	margin-bottom: 0;
}

.toggle-code-ext {
	display: inline-flex;
	align-items: center;
	gap: 0.375rem;
	font-size: 0.8125rem;
	color: var(--color-text-muted);
	cursor: pointer;
	white-space: nowrap;
}

.loading-inline {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 0.8125rem;
	color: var(--color-text-muted);
}

.spinner {
	width: 16px;
	height: 16px;
	border: 2px solid var(--color-border);
	border-top-color: var(--color-primary);
	border-radius: 50%;
	animation: spin 1s linear infinite;
	flex-shrink: 0;
}

@keyframes spin {
	to {
		transform: rotate(360deg);
	}
}

.error-banner {
	padding: 0.75rem 1.25rem;
	margin-bottom: 1rem;
	border: 1px solid #ef4444;
	border-radius: var(--radius-lg);
	color: #ef4444;
	font-size: 0.875rem;
}

.empty-state {
	text-align: center;
	padding: 4rem;
	color: var(--color-text-muted);
	background: var(--color-bg-card);
	border-radius: var(--radius-lg);
	border: 1px solid var(--color-border);
}

.team-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
	gap: 1.5rem;
	margin-bottom: 2rem;
}

.member-card {
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	padding: 1.25rem;
}

.member-card-link {
	display: block;
	text-decoration: none;
	color: inherit;
	transition:
		border-color 0.15s,
		box-shadow 0.15s;
}

.member-card-link:hover {
	border-color: var(--color-primary);
	box-shadow: 0 0 0 1px var(--color-primary);
}

.member-header {
	display: flex;
	align-items: center;
	gap: 1rem;
	margin-bottom: 1rem;
}

.avatar {
	width: 48px;
	height: 48px;
	border-radius: 50%;
}

.avatar-placeholder {
	width: 48px;
	height: 48px;
	border-radius: 50%;
	background: var(--color-primary);
	color: white;
	display: flex;
	align-items: center;
	justify-content: center;
	font-weight: 600;
	font-size: 1.25rem;
}

.member-info {
	flex: 1;
}

.member-name {
	font-size: 1rem;
	font-weight: 600;
}

.member-login {
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.member-stats {
	display: grid;
	grid-template-columns: repeat(4, 1fr);
	gap: 0.5rem;
	margin-bottom: 1rem;
}

.stat {
	text-align: center;
}

.stat-value {
	display: block;
	font-size: 1.25rem;
	font-weight: 600;
	color: var(--color-primary);
}

.stat-label {
	font-size: 0.625rem;
	color: var(--color-text-muted);
	text-transform: uppercase;
}

.code-stats {
	display: flex;
	justify-content: center;
	gap: 1rem;
	font-size: 0.875rem;
	font-family: var(--font-mono);
}

.additions {
	color: #22c55e;
}

.deletions {
	color: #ef4444;
}

.ext-stats {
	display: flex;
	flex-wrap: wrap;
	gap: 0.5rem;
	margin-top: 0.75rem;
	justify-content: center;
}

.ext-tag {
	display: inline-flex;
	align-items: center;
	gap: 0.25rem;
	padding: 0.125rem 0.5rem;
	background: var(--color-bg);
	border: 1px solid var(--color-border);
	border-radius: 4px;
	font-size: 0.75rem;
	font-family: var(--font-mono);
}

.ext-tag code {
	font-size: 0.75rem;
	font-weight: 600;
}

.ext-additions {
	color: #22c55e;
}

.ext-deletions {
	color: #ef4444;
}

.team-grid.is-stale {
	opacity: 0.5;
	pointer-events: none;
	transition: opacity 0.15s;
}

.skeleton {
	background: var(--color-border);
	border-radius: 4px;
	animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-avatar {
	width: 48px;
	height: 48px;
	border-radius: 50%;
	flex-shrink: 0;
}

.skeleton-line {
	height: 0.75rem;
}

.skeleton-line-name {
	width: 60%;
	margin-bottom: 0.5rem;
}

.skeleton-line-login {
	width: 40%;
}

.skeleton-stats {
	height: 2.75rem;
	margin-bottom: 1rem;
}

.skeleton-line-code {
	width: 40%;
	margin: 0 auto;
}

@keyframes pulse {
	50% {
		opacity: 0.4;
	}
}

@media (prefers-reduced-motion: reduce) {
	.skeleton {
		animation: none;
	}
}

.section {
	margin-top: 2rem;
}

.section h2 {
	font-size: 1.125rem;
	margin-bottom: 1rem;
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

.other-row {
	border-top: 2px solid var(--color-border);
	font-style: italic;
	color: var(--color-text-muted);
}

.table code {
	font-size: 0.875rem;
	padding: 0.125rem 0.375rem;
	background: var(--color-bg);
	border-radius: 4px;
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
}
</style>
