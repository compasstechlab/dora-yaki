<script lang="ts">
import { browser } from '$app/environment';
import CycleTimeChart from '$components/CycleTimeChart.svelte';
import MetricCard from '$components/MetricCard.svelte';
import { t } from '$i18n';
import { isCodeExtension, isConfigExtension } from '$lib/utils/extensions';
import {
	cycleTimeMetrics,
	dailyMetrics,
	dateRange,
	doraMetrics,
	formatHours,
	isLoading,
	loadMetrics,
	reviewMetrics,
} from '$stores/metrics';
import { currentRepositories, selectedRepositories } from '$stores/repositories';

let codeExtOnly = $state(true);
let configExtOnly = $state(false);

$effect(() => {
	if (browser && $dateRange) {
		loadMetrics($selectedRepositories, $dateRange.start, $dateRange.end);
	}
});

function updateDateRange() {
	loadMetrics($selectedRepositories, $dateRange.start, $dateRange.end);
}

let repoBadgeLabel = $derived(
	$selectedRepositories.length === 0
		? $t('common.allRepositories')
		: $currentRepositories.map((r) => r.fullName).join(', '),
);
</script>

<svelte:head>
	<title>{$t('metrics.pageTitle')}</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<h1>{$t('metrics.title')}</h1>
		<span class="repo-badge">{repoBadgeLabel}</span>
	</header>

	<!-- Date Range Filter -->
	<section class="section">
		<div class="filter-bar">
			<label class="filter-item">
				<span>{$t('common.startDate')}</span>
				<input type="date" bind:value={$dateRange.start} onchange={updateDateRange}>
			</label>
			<label class="filter-item">
				<span>{$t('common.endDate')}</span>
				<input type="date" bind:value={$dateRange.end} onchange={updateDateRange}>
			</label>
		</div>
	</section>

	{#if $isLoading}
		<div class="loading">{$t('common.loading')}</div>
	{:else}
		<!-- Cycle Time Analysis -->
		{#if $cycleTimeMetrics}
			<section class="section">
				<h2 class="section-title">{$t('metrics.cycleTimeAnalysis')}</h2>
				<div class="grid grid-4">
					<MetricCard
						title={$t('metrics.avgCycleTime')}
						value={formatHours($cycleTimeMetrics.avgCycleTime)}
						subtitle={$t('metrics.overall')}
						color="primary"
					/>
					<MetricCard
						title={$t('metrics.coding')}
						value={formatHours($cycleTimeMetrics.avgCodingTime)}
						subtitle={$t('metrics.codingSubtitle')}
						color="info"
					/>
					<MetricCard
						title={$t('metrics.pickup')}
						value={formatHours($cycleTimeMetrics.avgPickupTime)}
						subtitle={$t('metrics.pickupSubtitle')}
						color="warning"
					/>
					<MetricCard
						title={$t('metrics.review')}
						value={formatHours($cycleTimeMetrics.avgReviewTime)}
						subtitle={$t('metrics.reviewSubtitle')}
						color="success"
					/>
				</div>

				<div class="stats-row">
					<div class="stat">
						<span class="stat-label">{$t('metrics.median')}</span>
						<span class="stat-value">{formatHours($cycleTimeMetrics.medianCycleTime)}</span>
					</div>
					<div class="stat">
						<span class="stat-label">{$t('metrics.p90')}</span>
						<span class="stat-value">{formatHours($cycleTimeMetrics.p90CycleTime)}</span>
					</div>
					<div class="stat">
						<span class="stat-label">{$t('metrics.totalPRs')}</span>
						<span class="stat-value">{$cycleTimeMetrics.totalPRs}</span>
					</div>
				</div>

				{#if $dailyMetrics && $dailyMetrics.length > 0}
					<div class="card" style="margin-top: 1.5rem;">
						<CycleTimeChart data={$dailyMetrics} />
					</div>
				{/if}
			</section>

			<!-- Author Analysis -->
			{#if $cycleTimeMetrics.byAuthor && $cycleTimeMetrics.byAuthor.length > 0}
				<section class="section">
					<h2 class="section-title">{$t('metrics.authorAnalysis')}</h2>
					<div class="card">
						<table class="table">
							<thead>
								<tr>
									<th>{$t('metrics.author')}</th>
									<th>{$t('metrics.prCount')}</th>
									<th>{$t('metrics.avgCycleTimeCol')}</th>
									<th>{$t('metrics.addedLines')}</th>
									<th>{$t('metrics.deletedLines')}</th>
								</tr>
							</thead>
							<tbody>
								{#each $cycleTimeMetrics.byAuthor as author}
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
		{#if $doraMetrics}
			<section class="section">
				<h2 class="section-title">{$t('metrics.doraMetrics')}</h2>
				<div class="dora-grid">
					<div class="dora-card">
						<h3>{$t('metrics.deployFrequency')}</h3>
						<div class="dora-value">{$doraMetrics.deploymentFrequency}</div>
						<p class="dora-detail">
							{$t('metrics.deployCount', { count: $doraMetrics.deploymentCount, avg: $doraMetrics.avgDeploysPerDay.toFixed(2) })}
						</p>
					</div>
					<div class="dora-card">
						<h3>{$t('metrics.changeLeadTime')}</h3>
						<div class="dora-value">{formatHours($doraMetrics.avgLeadTime)}</div>
						<p class="dora-detail">
							{$t('metrics.leadTimeDetail', { median: formatHours($doraMetrics.medianLeadTime), p90: formatHours($doraMetrics.p90LeadTime) })}
						</p>
					</div>
					<div class="dora-card">
						<h3>{$t('metrics.changeFailureRate')}</h3>
						<div class="dora-value">{$doraMetrics.changeFailureRate.toFixed(1)}%</div>
						<p class="dora-detail">
							{$t('metrics.failureDetail', { failed: $doraMetrics.failedChanges, total: $doraMetrics.totalChanges })}
						</p>
					</div>
					<div class="dora-card">
						<h3>{$t('metrics.mttr')}</h3>
						<div class="dora-value">
							{$doraMetrics.avgMTTR > 0 ? formatHours($doraMetrics.avgMTTR) : 'N/A'}
						</div>
						<p class="dora-detail">
							{$t('metrics.incidentCount', { count: $doraMetrics.incidentCount })}
						</p>
					</div>
				</div>
			</section>
		{/if}

		<!-- Review Analysis -->
		{#if $reviewMetrics}
			<section class="section">
				<h2 class="section-title">{$t('metrics.reviewAnalysis')}</h2>
				<div class="grid grid-4">
					<MetricCard
						title={$t('metrics.totalReviews')}
						value={$reviewMetrics.totalReviews}
						color="primary"
					/>
					<MetricCard
						title={$t('metrics.firstReviewTime')}
						value={formatHours($reviewMetrics.avgTimeToFirstReview)}
						color="info"
					/>
					<MetricCard
						title={$t('metrics.approvalRate')}
						value="{$reviewMetrics.approvalRate.toFixed(1)}%"
						color="success"
					/>
					<MetricCard
						title={$t('metrics.changesRequestedRate')}
						value="{$reviewMetrics.changesRequestedRate.toFixed(1)}%"
						color="warning"
					/>
				</div>

				{#if $reviewMetrics.byReviewer && $reviewMetrics.byReviewer.length > 0}
					<div class="card" style="margin-top: 1.5rem;">
						<h3 class="card-title">{$t('metrics.reviewerStats')}</h3>
						<table class="table">
							<thead>
								<tr>
									<th>{$t('metrics.reviewer')}</th>
									<th>{$t('metrics.reviewCount')}</th>
									<th>{$t('metrics.commentCount')}</th>
									<th>{$t('metrics.approvalRate')}</th>
								</tr>
							</thead>
							<tbody>
								{#each $reviewMetrics.byReviewer as reviewer}
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
		{#if $cycleTimeMetrics?.byFileExtension && $cycleTimeMetrics.byFileExtension.length > 0}
			{@const filtered = $cycleTimeMetrics.byFileExtension.filter(e => {
				if (codeExtOnly && configExtOnly) return isCodeExtension(e.extension) || isConfigExtension(e.extension);
				if (codeExtOnly) return isCodeExtension(e.extension);
				if (configExtOnly) return isConfigExtension(e.extension);
				return true;
			})}
			{@const sorted = [...filtered].sort((a, b) => (b.additions + b.deletions) - (a.additions + a.deletions))}
			{@const top10 = sorted.slice(0, 10)}
			{@const rest = sorted.slice(10)}
			{@const other = rest.length > 0 ? {
				extension: $t('common.other'),
				additions: rest.reduce((s, r) => s + r.additions, 0),
				deletions: rest.reduce((s, r) => s + r.deletions, 0),
				files: rest.reduce((s, r) => s + r.files, 0),
				prCount: rest.reduce((s, r) => s + r.prCount, 0)
			} : null}
			<section class="section">
				<div class="section-header">
					<h2 class="section-title">{$t('metrics.fileExtChanges')}</h2>
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
								<th>{$t('metrics.extension')}</th>
								<th class="num">{$t('metrics.addedLinesShort')}</th>
								<th class="num">{$t('metrics.deletedLinesShort')}</th>
								<th class="num">{$t('metrics.fileCount')}</th>
								<th class="num">{$t('metrics.prCount')}</th>
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

.repo-badge {
	font-size: 0.875rem;
	padding: 0.25rem 0.75rem;
	background: var(--color-primary);
	color: white;
	border-radius: var(--radius-md);
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

.filter-bar {
	display: flex;
	gap: 1rem;
	padding: 1rem;
	background: var(--color-bg-card);
	border-radius: var(--radius-lg);
	border: 1px solid var(--color-border);
}

.filter-item {
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
}

.filter-item span {
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.filter-item input {
	padding: 0.5rem;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.loading {
	text-align: center;
	padding: 4rem;
	color: var(--color-text-muted);
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
