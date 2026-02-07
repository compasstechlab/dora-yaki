<script lang="ts">
import { onMount } from 'svelte';
import { api, type MemberPullRequest, type MemberReview, type MemberStats } from '$api/client';
import { browser } from '$app/environment';
import { page } from '$app/stores';
import { t } from '$i18n';
import MemberDailyChart from '$lib/components/MemberDailyChart.svelte';
import MemberWeeklyChart from '$lib/components/MemberWeeklyChart.svelte';
import { isCodeExtension, isConfigExtension } from '$lib/utils/extensions';
import { dateRange, formatHours } from '$stores/metrics';
import { selectedRepositories } from '$stores/repositories';

let stats = $state<MemberStats | null>(null);
let pullRequests = $state<MemberPullRequest[]>([]);
let reviews = $state<MemberReview[]>([]);
let isLoading = $state(true);
let error = $state('');
let showAllPRs = $state(false);
let codeExtOnly = $state(true);
let configExtOnly = $state(false);
const PR_DISPLAY_LIMIT = 15;

let visiblePRs = $derived(showAllPRs ? pullRequests : pullRequests.slice(0, PR_DISPLAY_LIMIT));
let hasMorePRs = $derived(pullRequests.length > PR_DISPLAY_LIMIT);

let memberId = $derived($page.params.id);

async function loadData() {
	if (!memberId) return;
	isLoading = true;
	error = '';
	showAllPRs = false;
	const repos = $selectedRepositories.length > 0 ? $selectedRepositories : undefined;
	try {
		const [s, prs, rvs] = await Promise.all([
			api.team.getMemberStats(memberId, repos, $dateRange.start, $dateRange.end),
			api.team.getMemberPullRequests(memberId, repos, $dateRange.start, $dateRange.end),
			api.team.getMemberReviews(memberId, repos, $dateRange.start, $dateRange.end),
		]);
		stats = s;
		pullRequests = prs;
		reviews = rvs;
	} catch (e) {
		console.error('Failed to load member data:', e);
		error = $t('memberDetail.loadError');
	} finally {
		isLoading = false;
	}
}

onMount(() => {
	loadData();
});

$effect(() => {
	if (browser && memberId && $selectedRepositories && $dateRange) {
		loadData();
	}
});

// Cycle time breakdown total (サイクルタイム内訳の合計)
let cycleTotal = $derived(
	stats ? stats.avgCodingTime + stats.avgPickupTime + stats.avgReviewTime + stats.avgMergeTime : 0,
);

function barPercent(value: number): number {
	if (cycleTotal <= 0) return 0;
	return (value / cycleTotal) * 100;
}

function formatDate(dateStr: string): string {
	return new Date(dateStr).toLocaleDateString('ja-JP', {
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
	});
}

function stateLabel(state: string): string {
	switch (state) {
		case 'merged':
			return 'Merged';
		case 'open':
			return 'Open';
		case 'closed':
			return 'Closed';
		default:
			return state;
	}
}

function stateClass(state: string): string {
	switch (state) {
		case 'merged':
			return 'badge-merged';
		case 'open':
			return 'badge-open';
		case 'closed':
			return 'badge-closed';
		default:
			return '';
	}
}
</script>

<svelte:head>
	<title>
		{stats
      ? stats.member.name || stats.member.login
      : $t("memberDetail.defaultTitle")}
		| DORA-yaki
	</title>
</svelte:head>

<div class="page">
	<a href="/team" class="back-link">{$t("memberDetail.backToTeam")}</a>

	{#if isLoading && !stats}
		<div class="loading">{$t("common.loading")}</div>
	{:else if error}
		<div class="error-state">{error}</div>
	{:else if stats}
		<!-- Header -->
		<header class="member-header">
			{#if stats.member.avatarUrl}
				<img src={stats.member.avatarUrl} alt={stats.member.login} class="avatar-lg">
			{:else}
				<div class="avatar-placeholder-lg">{stats.member.login[0].toUpperCase()}</div>
			{/if}
			<div>
				<h1 class="member-name">{stats.member.name || stats.member.login}</h1>
				<span class="member-login">@{stats.member.login}</span>
			</div>
		</header>

		<!-- Key Metrics -->
		<section class="metrics-grid">
			<div class="metric-card">
				<div class="metric-value">{stats.prsAuthored}</div>
				<div class="metric-label">{$t("memberDetail.prsCreated")}</div>
			</div>
			<div class="metric-card">
				<div class="metric-value">{stats.prsMerged}</div>
				<div class="metric-label">{$t("memberDetail.merged")}</div>
			</div>
			<div class="metric-card">
				<div class="metric-value">{stats.reviewsGiven}</div>
				<div class="metric-label">{$t("memberDetail.reviews")}</div>
			</div>
			<div class="metric-card">
				<div class="metric-value">
					{stats.avgCycleTime > 0 ? formatHours(stats.avgCycleTime) : "-"}
				</div>
				<div class="metric-label">{$t("memberDetail.avgCT")}</div>
			</div>
			<div class="metric-card">
				<div class="metric-value additions">+{stats.totalAdditions.toLocaleString()}</div>
				<div class="metric-label">{$t("memberDetail.addedLines")}</div>
			</div>
			<div class="metric-card">
				<div class="metric-value deletions">-{stats.totalDeletions.toLocaleString()}</div>
				<div class="metric-label">{$t("memberDetail.deletedLines")}</div>
			</div>
		</section>

		<!-- Cycle Time Breakdown -->
		{#if cycleTotal > 0}
			<section class="section">
				<h2>{$t("memberDetail.cycleTimeBreakdown")}</h2>
				<div class="card">
					<div class="cycle-bar-container">
						<div class="cycle-bar">
							<div
								class="cycle-segment coding"
								style="width: {barPercent(stats.avgCodingTime)}%"
								title="Coding: {formatHours(stats.avgCodingTime)}"
							></div>
							<div
								class="cycle-segment pickup"
								style="width: {barPercent(stats.avgPickupTime)}%"
								title="Pickup: {formatHours(stats.avgPickupTime)}"
							></div>
							<div
								class="cycle-segment review"
								style="width: {barPercent(stats.avgReviewTime)}%"
								title="Review: {formatHours(stats.avgReviewTime)}"
							></div>
							<div
								class="cycle-segment merge"
								style="width: {barPercent(stats.avgMergeTime)}%"
								title="Merge: {formatHours(stats.avgMergeTime)}"
							></div>
						</div>
						<div class="cycle-legend">
							<span class="legend-item">
								<span class="legend-dot coding"></span>
								Coding {formatHours(stats.avgCodingTime)}
							</span>
							<span class="legend-item">
								<span class="legend-dot pickup"></span>
								Pickup {formatHours(stats.avgPickupTime)}
							</span>
							<span class="legend-item">
								<span class="legend-dot review"></span>
								Review {formatHours(stats.avgReviewTime)}
							</span>
							<span class="legend-item">
								<span class="legend-dot merge"></span>
								Merge {formatHours(stats.avgMergeTime)}
							</span>
						</div>
					</div>
				</div>
			</section>
		{/if}

		<!-- Activity Charts -->
		{#if pullRequests.length > 0 || reviews.length > 0}
			<section class="section">
				<h2>{$t("memberDetail.dailyActivity")}</h2>
				<p class="chart-desc">{@html $t("memberDetail.dailyActivityDesc")}</p>
				<div class="card">
					<MemberDailyChart
						{pullRequests}
						{reviews}
						dateStart={$dateRange.start}
						dateEnd={$dateRange.end}
					/>
				</div>
			</section>

			<section class="section">
				<h2>{$t("memberDetail.weeklyActivity")}</h2>
				<div class="card">
					<MemberWeeklyChart
						{pullRequests}
						{reviews}
						dateStart={$dateRange.start}
						dateEnd={$dateRange.end}
					/>
				</div>
			</section>
		{/if}

		<!-- Review Stats -->
		<section class="section">
			<h2>{$t("memberDetail.reviewStats")}</h2>
			<div class="review-grid">
				<div class="metric-card">
					<div class="metric-value">{stats.reviewsGiven}</div>
					<div class="metric-label">{$t("memberDetail.totalReviews")}</div>
				</div>
				<div class="metric-card">
					<div class="metric-value">{stats.commentsGiven}</div>
					<div class="metric-label">{$t("memberDetail.commentCount")}</div>
				</div>
				<div class="metric-card">
					<div class="metric-value">
						{stats.approvalRate > 0 ? stats.approvalRate.toFixed(1) + "%" : "-"}
					</div>
					<div class="metric-label">{$t("memberDetail.approvalRate")}</div>
				</div>
				<div class="metric-card">
					<div class="metric-value">{stats.reviewsChangesRequested}</div>
					<div class="metric-label">{$t("memberDetail.changesRequested")}</div>
				</div>
			</div>
		</section>

		<!-- File Extension Stats -->
		{#if stats.byFileExtension && stats.byFileExtension.length > 0}
			{@const filteredExt = stats.byFileExtension.filter((e) => {
        if (codeExtOnly && configExtOnly)
          return isCodeExtension(e.extension) || isConfigExtension(e.extension);
        if (codeExtOnly) return isCodeExtension(e.extension);
        if (configExtOnly) return isConfigExtension(e.extension);
        return true;
      })}
			<section class="section">
				<div class="section-header">
					<h2>{$t("memberDetail.fileExtStats")}</h2>
					<label class="toggle-code-ext">
						<input type="checkbox" bind:checked={codeExtOnly}>
						<span>{$t("common.codeFilesOnly")}</span>
					</label>
					<label class="toggle-code-ext">
						<input type="checkbox" bind:checked={configExtOnly}>
						<span>{$t("common.configFilesOnly")}</span>
					</label>
				</div>
				<div class="card">
					<table class="table">
						<thead>
							<tr>
								<th>{$t("memberDetail.extension")}</th>
								<th class="num">{$t("memberDetail.prAdded")}</th>
								<th class="num">{$t("memberDetail.prDeleted")}</th>
								<th class="num">{$t("memberDetail.fileCount")}</th>
								<th class="num">{$t("memberDetail.prCount")}</th>
							</tr>
						</thead>
						<tbody>
							{#each filteredExt as ext}
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
						</tbody>
					</table>
				</div>
			</section>
		{/if}

		<!-- PR List -->
		{#if pullRequests.length > 0}
			<section class="section">
				<h2>{$t("memberDetail.prList", { count: pullRequests.length })}</h2>
				<div class="pr-list-container" class:collapsed={hasMorePRs && !showAllPRs}>
					<div class="card table-wrapper">
						<table class="table">
							<thead>
								<tr>
									<th>{$t("memberDetail.prTitle")}</th>
									<th>{$t("memberDetail.prState")}</th>
									<th>{$t("memberDetail.prRepo")}</th>
									<th class="num">{$t("memberDetail.prCreated")}</th>
									<th class="num">{$t("memberDetail.prMerged")}</th>
									<th class="num">{$t("memberDetail.prAdded")}</th>
									<th class="num">{$t("memberDetail.prDeleted")}</th>
									<th class="num">{$t("memberDetail.prCT")}</th>
								</tr>
							</thead>
							<tbody>
								{#each visiblePRs as pr}
									<tr>
										<td class="pr-title"><a
											href="https://github.com/{pr.repoName}/pull/{pr.number}"
											target="_blank"
											rel="noopener noreferrer"
											class="pr-link"
											title="#{pr.number} {pr.title}"
											onclick={(e) => {
                          e.stopPropagation();
                        }}
										>
											<span class="pr-number">#{pr.number}</span>
											{pr.title}
										</a></td>
										<td>
											<span class="badge {stateClass(pr.state)}">{stateLabel(pr.state)}</span>
										</td>
										<td class="repo-name" title={pr.repoName}>{pr.repoName}</td>
										<td class="num">{formatDate(pr.createdAt)}</td>
										<td class="num">{pr.mergedAt ? formatDate(pr.mergedAt) : "-"}</td>
										<td class="num ext-additions">+{pr.additions.toLocaleString()}</td>
										<td class="num ext-deletions">-{pr.deletions.toLocaleString()}</td>
										<td class="num">{pr.cycleTime > 0 ? formatHours(pr.cycleTime) : "-"}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
					{#if hasMorePRs && !showAllPRs}
						<div class="fade-overlay"></div>
					{/if}
				</div>
				{#if hasMorePRs}
					<button class="show-more-btn" onclick={() => (showAllPRs = !showAllPRs)}>
						{showAllPRs
              ? $t("common.showLess")
              : $t("common.showMore", {
                  remaining: pullRequests.length - PR_DISPLAY_LIMIT,
                })}
					</button>
				{/if}
			</section>
		{/if}
	{/if}
</div>

<style>
.page {
	max-width: 1200px;
	margin: 0 auto;
}

.back-link {
	display: inline-block;
	margin-bottom: 1.5rem;
	color: var(--color-primary);
	text-decoration: none;
	font-size: 0.875rem;
}

.back-link:hover {
	text-decoration: underline;
}

.loading,
.error-state {
	text-align: center;
	padding: 4rem;
	color: var(--color-text-muted);
	background: var(--color-bg-card);
	border-radius: var(--radius-lg);
	border: 1px solid var(--color-border);
}

/* Header */
.member-header {
	display: flex;
	align-items: center;
	gap: 1.25rem;
	margin-bottom: 2rem;
}

.avatar-lg {
	width: 64px;
	height: 64px;
	border-radius: 50%;
}

.avatar-placeholder-lg {
	width: 64px;
	height: 64px;
	border-radius: 50%;
	background: var(--color-primary);
	color: white;
	display: flex;
	align-items: center;
	justify-content: center;
	font-weight: 600;
	font-size: 1.5rem;
}

.member-name {
	font-size: 1.5rem;
	font-weight: 700;
	margin: 0;
}

.member-login {
	font-size: 0.875rem;
	color: var(--color-text-muted);
}

/* Metrics Grid */
.metrics-grid {
	display: grid;
	grid-template-columns: repeat(6, 1fr);
	gap: 1rem;
	margin-bottom: 2rem;
}

.review-grid {
	display: grid;
	grid-template-columns: repeat(4, 1fr);
	gap: 1rem;
}

.metric-card {
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	padding: 1.25rem;
	text-align: center;
}

.metric-value {
	font-size: 1.5rem;
	font-weight: 700;
	color: var(--color-primary);
}

.metric-label {
	font-size: 0.75rem;
	color: var(--color-text-muted);
	text-transform: uppercase;
	margin-top: 0.25rem;
}

.additions {
	color: #22c55e;
}

.deletions {
	color: #ef4444;
}

/* Section */
.section {
	margin-top: 2rem;
}

.section h2 {
	font-size: 1.125rem;
	margin-bottom: 0.5rem;
}

.section-header {
	display: flex;
	align-items: center;
	gap: 1rem;
	margin-bottom: 0.5rem;
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

.chart-desc {
	font-size: 0.75rem;
	color: var(--color-text-muted);
	margin-bottom: 0.75rem;
}

:global(.sat-label) {
	color: #06b6d4;
	font-weight: 600;
}

:global(.sun-label) {
	color: #f87171;
	font-weight: 600;
}

.card {
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	padding: 1.25rem;
}

/* Cycle Time Bar */
.cycle-bar-container {
	display: flex;
	flex-direction: column;
	gap: 0.75rem;
}

.cycle-bar {
	display: flex;
	height: 32px;
	border-radius: 6px;
	overflow: hidden;
}

.cycle-segment {
	min-width: 2px;
	transition: width 0.3s ease;
}

.cycle-segment.coding {
	background: #3b82f6;
}

.cycle-segment.pickup {
	background: #f59e0b;
}

.cycle-segment.review {
	background: #8b5cf6;
}

.cycle-segment.merge {
	background: #22c55e;
}

.cycle-legend {
	display: flex;
	gap: 1.5rem;
	flex-wrap: wrap;
}

.legend-item {
	display: flex;
	align-items: center;
	gap: 0.375rem;
	font-size: 0.8125rem;
}

.legend-dot {
	width: 10px;
	height: 10px;
	border-radius: 2px;
}

.legend-dot.coding {
	background: #3b82f6;
}

.legend-dot.pickup {
	background: #f59e0b;
}

.legend-dot.review {
	background: #8b5cf6;
}

.legend-dot.merge {
	background: #22c55e;
}

/* Table */
.table-wrapper {
	overflow-x: auto;
	padding: 0;
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

.pr-title {
	max-width: 320px;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.pr-link {
	color: inherit;
	text-decoration: none;
	display: block;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.pr-link:hover {
	color: var(--color-primary);
}

.pr-number {
	color: var(--color-text-muted);
	margin-right: 0.375rem;
	font-size: 0.8125rem;
}

.repo-name {
	font-size: 0.8125rem;
	color: var(--color-text-muted);
	max-width: 160px;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.badge {
	display: inline-block;
	padding: 0.125rem 0.5rem;
	border-radius: 9999px;
	font-size: 0.75rem;
	font-weight: 600;
}

.badge-merged {
	background: rgba(139, 92, 246, 0.15);
	color: #8b5cf6;
}

.badge-open {
	background: rgba(34, 197, 94, 0.15);
	color: #22c55e;
}

.badge-closed {
	background: rgba(239, 68, 68, 0.15);
	color: #ef4444;
}

/* PR List Collapse */
.pr-list-container {
	position: relative;
}

.pr-list-container.collapsed {
	max-height: 720px;
	overflow: hidden;
}

.fade-overlay {
	position: absolute;
	bottom: 0;
	left: 0;
	right: 0;
	height: 80px;
	background: linear-gradient(to bottom, transparent, var(--color-bg));
	pointer-events: none;
	border-radius: 0 0 var(--radius-lg) var(--radius-lg);
}

.show-more-btn {
	display: block;
	width: 100%;
	margin-top: 0.5rem;
	padding: 0.625rem;
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	color: var(--color-primary);
	font-size: 0.875rem;
	font-weight: 600;
	cursor: pointer;
	transition:
		background 0.15s,
		border-color 0.15s;
}

.show-more-btn:hover {
	background: var(--color-bg);
	border-color: var(--color-primary);
}

.ext-additions {
	color: #22c55e;
}

.ext-deletions {
	color: #ef4444;
}

.table code {
	font-size: 0.875rem;
	padding: 0.125rem 0.375rem;
	background: var(--color-bg);
	border-radius: 4px;
}

@media (max-width: 1024px) {
	.metrics-grid {
		grid-template-columns: repeat(3, 1fr);
	}
}

@media (max-width: 640px) {
	.metrics-grid {
		grid-template-columns: repeat(2, 1fr);
	}

	.review-grid {
		grid-template-columns: repeat(2, 1fr);
	}
}
</style>
