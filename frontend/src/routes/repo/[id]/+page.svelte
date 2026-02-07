<script lang="ts">
import { onMount } from 'svelte';
import {
	api,
	type CycleTimeMetrics,
	type DailyMetrics,
	type DORAMetrics,
	type MemberPullRequest,
	type ProductivityScore,
	type Repository,
	type ReviewMetrics,
} from '$api/client';
import { browser } from '$app/environment';
import { page } from '$app/stores';
import CycleTimeChart from '$components/CycleTimeChart.svelte';
import MetricCard from '$components/MetricCard.svelte';
import PRActivityChart from '$components/PRActivityChart.svelte';
import ScoreGauge from '$components/ScoreGauge.svelte';
import { t } from '$i18n';
import { isCodeExtension, isConfigExtension } from '$lib/utils/extensions';
import { dateRange, formatHours } from '$stores/metrics';
import { repositories } from '$stores/repositories';

let repo = $state<Repository | null>(null);
let cycleTime = $state<CycleTimeMetrics | null>(null);
let reviews = $state<ReviewMetrics | null>(null);
let dora = $state<DORAMetrics | null>(null);
let score = $state<ProductivityScore | null>(null);
let daily = $state<DailyMetrics[]>([]);
let pullRequests = $state<MemberPullRequest[]>([]);
let isLoading = $state(true);
let showAllPRs = $state(false);
let codeExtOnly = $state(true);
let configExtOnly = $state(false);
const PR_DISPLAY_LIMIT = 15;

let visiblePRs = $derived(showAllPRs ? pullRequests : pullRequests.slice(0, PR_DISPLAY_LIMIT));
let hasMorePRs = $derived(pullRequests.length > PR_DISPLAY_LIMIT);

let repoId = $derived($page.params.id);

onMount(() => {
	repo = $repositories.find((r) => r.id === repoId) || null;
});

$effect(() => {
	if (browser && repoId && $dateRange) {
		loadData();
	}
});

async function loadData() {
	isLoading = true;
	showAllPRs = false;
	try {
		const id = repoId;
		if (!id) return;
		const [ct, rv, dr, sc, dl, prs] = await Promise.all([
			api.metrics.cycleTime([id], $dateRange.start, $dateRange.end),
			api.metrics.reviews([id], $dateRange.start, $dateRange.end),
			api.metrics.dora([id], $dateRange.start, $dateRange.end),
			api.metrics.productivityScore([id], $dateRange.start, $dateRange.end),
			api.metrics.daily([id], $dateRange.start, $dateRange.end),
			api.metrics.pullRequests([id], $dateRange.start, $dateRange.end),
		]);
		cycleTime = ct;
		reviews = rv;
		dora = dr;
		score = sc;
		daily = dl;
		pullRequests = prs;
	} catch (e) {
		console.error('Failed to load repo metrics:', e);
	} finally {
		isLoading = false;
	}
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
	<title>{repo?.fullName || $t("repoDetail.defaultTitle")} | DORA-yaki</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<a href="/repo" class="back-link">{$t("repoDetail.backToList")}</a>
		<h1>{repo?.fullName || repoId}</h1>
	</header>

	{#if isLoading}
		<div class="loading">
			<div class="spinner"></div>
			<p>{$t("common.loadingMetrics")}</p>
		</div>
	{:else}
		<!-- Productivity Score -->
		{#if score}
			<section class="section">
				<h2 class="section-title">{$t("repoDetail.productivityScore")}</h2>
				<div class="score-overview">
					<div class="score-main">
						<ScoreGauge
							score={score.overallScore}
							label={$t("repoDetail.overallScore")}
							size="lg"
						/>
					</div>
					<div class="score-components">
						{#each score.componentScores || [] as component}
							<div class="score-component">
								<ScoreGauge score={component.score} label={component.name} size="sm" />
								<span class="component-name">{component.name}</span>
							</div>
						{/each}
					</div>
				</div>

				{#if score.recommendations && score.recommendations.length > 0}
					<div class="recommendations">
						<h3>{$t("repoDetail.recommendations")}</h3>
						<ul>
							{#each score.recommendations as rec}
								<li>{rec}</li>
							{/each}
						</ul>
					</div>
				{/if}
			</section>
		{/if}

		<!-- Cycle Time Analysis -->
		{#if cycleTime}
			<section class="section">
				<h2 class="section-title">{$t("repoDetail.cycleTimeAnalysis")}</h2>
				<div class="grid grid-4">
					<MetricCard
						title={$t("repoDetail.avgCycleTime")}
						value={formatHours(cycleTime.avgCycleTime)}
						subtitle={$t("repoDetail.overall")}
						color="primary"
					/>
					<MetricCard
						title={$t("repoDetail.coding")}
						value={formatHours(cycleTime.avgCodingTime)}
						subtitle={$t("repoDetail.codingSubtitle")}
						color="info"
					/>
					<MetricCard
						title={$t("repoDetail.pickup")}
						value={formatHours(cycleTime.avgPickupTime)}
						subtitle={$t("repoDetail.pickupSubtitle")}
						color="warning"
					/>
					<MetricCard
						title={$t("repoDetail.review")}
						value={formatHours(cycleTime.avgReviewTime)}
						subtitle={$t("repoDetail.reviewSubtitle")}
						color="success"
					/>
				</div>

				<div class="stats-row">
					<div class="stat">
						<span class="stat-label">{$t("repoDetail.median")}</span>
						<span class="stat-value">{formatHours(cycleTime.medianCycleTime)}</span>
					</div>
					<div class="stat">
						<span class="stat-label">{$t("repoDetail.p90")}</span>
						<span class="stat-value">{formatHours(cycleTime.p90CycleTime)}</span>
					</div>
					<div class="stat">
						<span class="stat-label">{$t("repoDetail.totalPRs")}</span>
						<span class="stat-value">{cycleTime.totalPRs}</span>
					</div>
				</div>

				{#if daily && daily.length > 0}
					<div class="card" style="margin-top: 1.5rem;">
						<CycleTimeChart data={daily} />
					</div>
				{/if}
			</section>

			<!-- Author Analysis -->
			{#if cycleTime.byAuthor && cycleTime.byAuthor.length > 0}
				<section class="section">
					<h2 class="section-title">{$t("repoDetail.authorAnalysis")}</h2>
					<div class="card">
						<table class="table">
							<thead>
								<tr>
									<th>{$t("repoDetail.author")}</th>
									<th>{$t("repoDetail.prCount")}</th>
									<th>{$t("repoDetail.avgCycleTimeCol")}</th>
									<th>{$t("repoDetail.addedLines")}</th>
									<th>{$t("repoDetail.deletedLines")}</th>
								</tr>
							</thead>
							<tbody>
								{#each cycleTime.byAuthor as author}
									<tr>
										<td>{author.author}</td>
										<td>{author.prCount}</td>
										<td>{formatHours(author.avgCycleTime)}</td>
										<td class="additions">+{author.additions.toLocaleString()}</td>
										<td class="deletions">-{author.deletions.toLocaleString()}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</section>
			{/if}
		{/if}

		<!-- DORA Metrics -->
		{#if dora}
			<section class="section">
				<h2 class="section-title">{$t("repoDetail.doraMetrics")}</h2>
				<div class="dora-grid">
					<div class="dora-card">
						<h3>{$t("repoDetail.deployFrequency")}</h3>
						<div class="dora-value">
							{$t(
                `deployFrequency.${dora.deploymentFrequency === "daily" ? "daily" : dora.deploymentFrequency === "weekly" ? "weekly" : dora.deploymentFrequency === "monthly" ? "monthly" : "low"}`,
              )}
						</div>
						<p class="dora-detail">
							{$t("repoDetail.deployCount", {
                count: dora.deploymentCount,
                avg: dora.avgDeploysPerDay.toFixed(2),
              })}
						</p>
					</div>
					<div class="dora-card">
						<h3>{$t("repoDetail.changeLeadTime")}</h3>
						<div class="dora-value">{formatHours(dora.avgLeadTime)}</div>
						<p class="dora-detail">
							{$t("repoDetail.leadTimeDetail", {
                median: formatHours(dora.medianLeadTime),
                p90: formatHours(dora.p90LeadTime),
              })}
						</p>
					</div>
					<div class="dora-card">
						<h3>{$t("repoDetail.changeFailureRate")}</h3>
						<div class="dora-value">{dora.changeFailureRate.toFixed(1)}%</div>
						<p class="dora-detail">
							{$t("repoDetail.failureDetail", {
                failed: dora.failedChanges,
                total: dora.totalChanges,
              })}
						</p>
					</div>
					<div class="dora-card">
						<h3>{$t("repoDetail.mttr")}</h3>
						<div class="dora-value">{dora.avgMTTR > 0 ? formatHours(dora.avgMTTR) : "N/A"}</div>
						<p class="dora-detail">
							{$t("repoDetail.incidentCount", { count: dora.incidentCount })}
						</p>
					</div>
				</div>
			</section>
		{/if}

		<!-- PR Activity Chart -->
		{#if daily && daily.length > 0}
			<section class="section">
				<h2 class="section-title">{$t("repoDetail.prActivity")}</h2>
				<div class="card">
					<PRActivityChart data={daily} />
				</div>
			</section>
		{/if}

		<!-- Review Analysis -->
		{#if reviews}
			<section class="section">
				<h2 class="section-title">{$t("repoDetail.reviewAnalysis")}</h2>
				<div class="grid grid-4">
					<MetricCard
						title={$t("repoDetail.totalReviews")}
						value={reviews.totalReviews}
						color="primary"
					/>
					<MetricCard
						title={$t("repoDetail.firstReviewTime")}
						value={formatHours(reviews.avgTimeToFirstReview)}
						color="info"
					/>
					<MetricCard
						title={$t("repoDetail.approvalRate")}
						value="{reviews.approvalRate.toFixed(1)}%"
						color="success"
					/>
					<MetricCard
						title={$t("repoDetail.changesRequestedRate")}
						value="{reviews.changesRequestedRate.toFixed(1)}%"
						color="warning"
					/>
				</div>

				{#if reviews.byReviewer && reviews.byReviewer.length > 0}
					<div class="card" style="margin-top: 1.5rem;">
						<h3 class="card-title">{$t("repoDetail.reviewerStats")}</h3>
						<table class="table">
							<thead>
								<tr>
									<th>{$t("repoDetail.reviewer")}</th>
									<th>{$t("repoDetail.reviewCount")}</th>
									<th>{$t("repoDetail.commentCount")}</th>
									<th>{$t("repoDetail.approvalRate")}</th>
								</tr>
							</thead>
							<tbody>
								{#each reviews.byReviewer as reviewer}
									<tr>
										<td>{reviewer.reviewer}</td>
										<td>{reviewer.reviewCount}</td>
										<td>{reviewer.commentCount}</td>
										<td>{reviewer.approvalRate.toFixed(1)}%</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</section>
		{/if}

		<!-- File Extension Stats -->
		{#if cycleTime?.byFileExtension && cycleTime.byFileExtension.length > 0}
			{@const filtered = cycleTime.byFileExtension.filter((e) => {
        if (codeExtOnly && configExtOnly)
          return isCodeExtension(e.extension) || isConfigExtension(e.extension);
        if (codeExtOnly) return isCodeExtension(e.extension);
        if (configExtOnly) return isConfigExtension(e.extension);
        return true;
      })}
			{@const sorted = [...filtered].sort(
        (a, b) => b.additions + b.deletions - (a.additions + a.deletions),
      )}
			{@const top10 = sorted.slice(0, 10)}
			{@const rest = sorted.slice(10)}
			{@const other =
        rest.length > 0
          ? {
              extension: $t("common.other"),
              additions: rest.reduce((s, r) => s + r.additions, 0),
              deletions: rest.reduce((s, r) => s + r.deletions, 0),
              files: rest.reduce((s, r) => s + r.files, 0),
              prCount: rest.reduce((s, r) => s + r.prCount, 0),
            }
          : null}
			<section class="section">
				<div class="section-header">
					<h2 class="section-title">{$t("repoDetail.fileExtChanges")}</h2>
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
								<th>{$t("repoDetail.extension")}</th>
								<th class="num">{$t("repoDetail.addedLinesShort")}</th>
								<th class="num">{$t("repoDetail.deletedLinesShort")}</th>
								<th class="num">{$t("repoDetail.fileCount")}</th>
								<th class="num">{$t("repoDetail.prCountShort")}</th>
							</tr>
						</thead>
						<tbody>
							{#each top10 as ext}
								<tr>
									<td>
										<code>{ext.extension}</code>
									</td>
									<td class="num additions">+{ext.additions.toLocaleString()}</td>
									<td class="num deletions">-{ext.deletions.toLocaleString()}</td>
									<td class="num">{ext.files.toLocaleString()}</td>
									<td class="num">{ext.prCount}</td>
								</tr>
							{/each}
							{#if other}
								<tr class="other-row">
									<td>{other.extension}</td>
									<td class="num additions">+{other.additions.toLocaleString()}</td>
									<td class="num deletions">-{other.deletions.toLocaleString()}</td>
									<td class="num">{other.files.toLocaleString()}</td>
									<td class="num">{other.prCount}</td>
								</tr>
							{/if}
						</tbody>
					</table>
				</div>
			</section>
		{/if}

		<!-- PR List -->
		{#if pullRequests.length > 0}
			<section class="section">
				<h2 class="section-title">{$t("memberDetail.prList", { count: pullRequests.length })}</h2>
				<div class="pr-list-container" class:collapsed={hasMorePRs && !showAllPRs}>
					<div class="card table-wrapper">
						<table class="table">
							<thead>
								<tr>
									<th>{$t("memberDetail.prTitle")}</th>
									<th>{$t("repoDetail.author")}</th>
									<th>{$t("memberDetail.prState")}</th>
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
										<td class="pr-title">
											<a
												href="https://github.com/{pr.repoName}/pull/{pr.number}"
												target="_blank"
												rel="noopener noreferrer"
												class="pr-link"
												title="#{pr.number} {pr.title}"
											>
												<span class="pr-number">#{pr.number}</span>
												{pr.title}
											</a>
										</td>
										<td class="author-name">{pr.author || "-"}</td>
										<td>
											<span class="badge {stateClass(pr.state)}">{stateLabel(pr.state)}</span>
										</td>
										<td class="num">{formatDate(pr.createdAt)}</td>
										<td class="num">{pr.mergedAt ? formatDate(pr.mergedAt) : "-"}</td>
										<td class="num additions">+{pr.additions.toLocaleString()}</td>
										<td class="num deletions">-{pr.deletions.toLocaleString()}</td>
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
	max-width: 1400px;
	margin: 0 auto;
}

.page-header {
	display: flex;
	align-items: center;
	gap: 1rem;
	margin-bottom: 2rem;
}

.page-header h1 {
	font-size: 1.75rem;
}

.back-link {
	font-size: 0.875rem;
	color: var(--color-primary);
	text-decoration: none;
}

.back-link:hover {
	text-decoration: underline;
}

.section {
	margin-bottom: 2rem;
}

.section-title {
	font-size: 1.125rem;
	margin-bottom: 1rem;
}

.section-header {
	display: flex;
	align-items: center;
	gap: 1rem;
	margin-bottom: 1rem;
}

.section-header .section-title {
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

.loading {
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	padding: 4rem;
}

.spinner {
	width: 40px;
	height: 40px;
	border: 3px solid var(--color-border);
	border-top-color: var(--color-primary);
	border-radius: 50%;
	animation: spin 1s linear infinite;
}

@keyframes spin {
	to {
		transform: rotate(360deg);
	}
}

.score-overview {
	display: flex;
	gap: 2rem;
	padding: 2rem;
	background: var(--color-bg-card);
	border-radius: var(--radius-lg);
	border: 1px solid var(--color-border);
}

.score-main {
	display: flex;
	align-items: center;
	justify-content: center;
	padding: 1rem;
}

.score-components {
	flex: 1;
	display: grid;
	grid-template-columns: repeat(4, 1fr);
	gap: 1.5rem;
}

.score-component {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 0.5rem;
}

.component-name {
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.recommendations {
	margin-top: 1.5rem;
	padding: 1rem;
	background: rgba(59, 130, 246, 0.05);
	border-radius: var(--radius-md);
	border-left: 3px solid var(--color-primary);
}

.recommendations h3 {
	font-size: 0.875rem;
	margin-bottom: 0.5rem;
}

.recommendations ul {
	margin: 0;
	padding-left: 1.25rem;
}

.recommendations li {
	font-size: 0.875rem;
	color: var(--color-text-muted);
	margin-bottom: 0.25rem;
}

.stats-row {
	display: flex;
	gap: 2rem;
	margin-top: 1.5rem;
	padding: 1rem;
	background: var(--color-bg-card);
	border-radius: var(--radius-md);
	border: 1px solid var(--color-border);
}

.stat {
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
}

.stat-label {
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.stat-value {
	font-size: 1.25rem;
	font-weight: 600;
}

.card-title {
	font-size: 0.875rem;
	font-weight: 600;
	margin-bottom: 1rem;
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
	color: var(--color-text-muted);
}

.additions {
	color: #22c55e;
}

.deletions {
	color: #ef4444;
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

/* PR List */
.table-wrapper {
	overflow-x: auto;
	padding: 0;
}

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

.author-name {
	font-size: 0.8125rem;
	color: var(--color-text-muted);
	max-width: 120px;
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

.dora-grid {
	display: grid;
	grid-template-columns: repeat(4, 1fr);
	gap: 1rem;
}

.dora-card {
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	padding: 1.25rem;
}

.dora-card h3 {
	font-size: 0.875rem;
	color: var(--color-text-muted);
	margin-bottom: 0.5rem;
}

.dora-value {
	font-size: 1.5rem;
	font-weight: 700;
	color: var(--color-primary);
}

.dora-detail {
	font-size: 0.75rem;
	color: var(--color-text-muted);
	margin-top: 0.5rem;
}

@media (max-width: 1024px) {
	.score-overview {
		flex-direction: column;
	}

	.score-components {
		grid-template-columns: repeat(2, 1fr);
	}

	.dora-grid {
		grid-template-columns: repeat(2, 1fr);
	}
}

@media (max-width: 640px) {
	.dora-grid {
		grid-template-columns: 1fr;
	}
}
</style>
