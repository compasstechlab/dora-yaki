<script lang="ts">
import { onMount } from 'svelte';
import { api, type FileExtensionMetrics, type MemberStats, type TeamMember } from '$api/client';
import { browser } from '$app/environment';
import { t } from '$i18n';
import { isCodeExtension, isConfigExtension } from '$lib/utils/extensions';
import { dateRange, formatHours } from '$stores/metrics';
import { selectedRepositories } from '$stores/repositories';

let members = $state<TeamMember[]>([]);
let memberStats = $state<Record<string, MemberStats>>({});
let isLoading = $state(false);
let membersLoaded = $state(false);
let hideInactive = $state(true);
let codeExtOnly = $state(true);
let configExtOnly = $state(false);

function isActiveMember(stats: MemberStats): boolean {
	return (
		stats.prsAuthored > 0 ||
		stats.prsMerged > 0 ||
		stats.reviewsGiven > 0 ||
		stats.totalAdditions > 0 ||
		stats.totalDeletions > 0
	);
}

let statsLoaded = $derived(Object.keys(memberStats).length >= members.length && members.length > 0);

let visibleMembers = $derived(
	members.filter((m) => {
		if (!hideInactive) return true;
		const stats = memberStats[m.id];
		if (!stats) return !statsLoaded;
		return isActiveMember(stats);
	}),
);

// Aggregate file extension stats across all team members
let teamFileExtStats = $derived.by(() => {
	const map = new Map<string, FileExtensionMetrics>();
	for (const stats of Object.values(memberStats)) {
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

onMount(async () => {
	await loadMembers();
	membersLoaded = true;
});

async function loadMembers() {
	try {
		members = await api.team.listMembers();
	} catch (error) {
		console.error('Failed to load team members:', error);
	}
}

async function loadAllMemberStats() {
	if (members.length === 0) return;
	isLoading = true;
	// Clear old stats on reload and show new data incrementally
	memberStats = {};
	const repos = $selectedRepositories.length > 0 ? $selectedRepositories : undefined;
	await Promise.all(
		members.map(async (m) => {
			try {
				const stats = await api.team.getMemberStats(m.id, repos, $dateRange.start, $dateRange.end);
				memberStats[m.id] = stats;
			} catch (error) {
				console.error('Failed to load member stats:', error);
			}
		}),
	);
	isLoading = false;
}

$effect(() => {
	if (browser && membersLoaded && $selectedRepositories && $dateRange) {
		loadAllMemberStats();
	}
});
</script>

<svelte:head>
	<title>{$t('team.pageTitle')}</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<h1>{$t('team.title')}</h1>
		<label class="toggle-inactive">
			<input type="checkbox" bind:checked={hideInactive}>
			<span>{$t('common.activeOnly')}</span>
		</label>
	</header>

	{#if members.length === 0}
		<div class="empty-state">
			<p>{$t('team.noMembers')}</p>
		</div>
	{:else}
		{#if isLoading}
			<div class="loading-bar">
				<div class="spinner"></div>
				<span>{$t('common.loadingStats')}</span>
				{#if members.length > 0}
					<span class="loading-progress"
						>({Object.keys(memberStats).length} / {members.length})</span
					>
				{/if}
			</div>
		{/if}
		{#if statsLoaded && visibleMembers.length === 0 && hideInactive}
			<div class="empty-state">
				<p>{$t('team.noActiveMembers')}</p>
			</div>
		{/if}
		<div class="team-grid">
			{#each visibleMembers as member}
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

					{#if memberStats[member.id]}
						<div class="member-stats">
							<div class="stat">
								<span class="stat-value">{memberStats[member.id].prsAuthored}</span>
								<span class="stat-label">{$t('team.prsCreated')}</span>
							</div>
							<div class="stat">
								<span class="stat-value">{memberStats[member.id].prsMerged}</span>
								<span class="stat-label">{$t('team.merged')}</span>
							</div>
							<div class="stat">
								<span class="stat-value">{memberStats[member.id].reviewsGiven}</span>
								<span class="stat-label">{$t('team.reviews')}</span>
							</div>
							<div class="stat">
								<span class="stat-value">
									{memberStats[member.id].avgCycleTime > 0
                    ? formatHours(memberStats[member.id].avgCycleTime)
                    : "-"}
								</span>
								<span class="stat-label">{$t('team.avgCT')}</span>
							</div>
						</div>

						<div class="code-stats">
							<span class="additions"
								>+{memberStats[member.id].totalAdditions.toLocaleString()}</span
							>
							<span class="deletions"
								>-{memberStats[member.id].totalDeletions.toLocaleString()}</span
							>
						</div>

						{#if memberStats[member.id].byFileExtension && memberStats[member.id].byFileExtension!.length > 0}
							<div class="ext-stats">
								{#each memberStats[member.id].byFileExtension!.slice(0, 3) as ext}
									<span class="ext-tag">
										<code>{ext.extension}</code>
										<span class="ext-additions">+{ext.additions.toLocaleString()}</span>
										<span class="ext-deletions">-{ext.deletions.toLocaleString()}</span>
									</span>
								{/each}
							</div>
						{/if}
					{:else}
						<div class="loading-stats">{$t('common.loadingStats')}</div>
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
					<div class="summary-value">
						{Object.values(memberStats).reduce(
              (sum, s) => sum + s.prsAuthored,
              0,
            )}
					</div>
				</div>
				<div class="summary-card">
					<h3>{$t('team.totalReviews')}</h3>
					<div class="summary-value">
						{Object.values(memberStats).reduce(
              (sum, s) => sum + s.reviewsGiven,
              0,
            )}
					</div>
				</div>
				<div class="summary-card">
					<h3>{$t('team.activeMembers')}</h3>
					<div class="summary-value">
						{Object.values(memberStats).filter(
              (s) => s.prsAuthored > 0 || s.reviewsGiven > 0,
            ).length}
					</div>
				</div>
				<div class="summary-card">
					<h3>{$t('team.totalCodeChanges')}</h3>
					<div class="summary-value">
						{Object.values(memberStats)
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

.loading-bar {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 0.75rem 1.25rem;
	margin-bottom: 1rem;
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	font-size: 0.875rem;
	color: var(--color-text-muted);
}

.spinner {
	width: 20px;
	height: 20px;
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

.loading-progress {
	font-variant-numeric: tabular-nums;
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

.loading-stats {
	text-align: center;
	padding: 1rem;
	color: var(--color-text-muted);
	font-size: 0.875rem;
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
