<script lang="ts">
import { api } from '$api/client';
import { browser } from '$app/environment';
import CycleTimeChart from '$components/CycleTimeChart.svelte';
import MetricCard from '$components/MetricCard.svelte';
import PRActivityChart from '$components/PRActivityChart.svelte';
import ScoreGauge from '$components/ScoreGauge.svelte';
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
	productivityScore,
	reviewMetrics,
} from '$stores/metrics';
import { currentRepositories, selectedRepositories } from '$stores/repositories';

let codeExtOnly = $state(true);
let configExtOnly = $state(false);

$effect(() => {
	if (browser && $dateRange) loadMetrics($selectedRepositories, $dateRange.start, $dateRange.end);
});

let repoBadgeLabel = $derived(
	$selectedRepositories.length === 0
		? $t('common.allRepositories')
		: $currentRepositories.map((r) => r.fullName).join(', '),
);

// Invalidate all caches and refresh (全キャッシュ削除してリフレッシュ)
async function handleRefresh() {
	await api.cache.invalidate();
	loadMetrics($selectedRepositories, $dateRange.start, $dateRange.end, true);
}
</script>

<svelte:head>
	<title>{$t('dashboard.pageTitle')}</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<h1>{$t('dashboard.title')}</h1>
		<span class="repo-badge">{repoBadgeLabel}</span>
		<button
			class="refresh-btn"
			onclick={handleRefresh}
			disabled={$isLoading}
			title={$t('common.refreshTitle')}
		>
			<svg
				class="refresh-icon"
				class:spinning={$isLoading}
				viewBox="0 0 24 24"
				width="16"
				height="16"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
			>
				<path d="M23 4v6h-6M1 20v-6h6" />
				<path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
			</svg>
			{$t('common.refresh')}
		</button>
	</header>

	{#if $isLoading}
		<div class="loading">
			<div class="spinner"></div>
			<p>{$t('common.loadingMetrics')}</p>
		</div>
	{:else}
		<!-- Productivity Score -->
		{#if $productivityScore}
			<section class="section">
				<h2 class="section-title">{$t('dashboard.productivityScore')}</h2>
				<div class="score-overview">
					<div class="score-main">
						<ScoreGauge
							score={$productivityScore.overallScore}
							label={$t('dashboard.overallScore')}
							size="lg"
						/>
					</div>
					<div class="score-components">
						{#each $productivityScore.componentScores || [] as component}
							<div class="score-component">
								<ScoreGauge score={component.score} label={component.name} size="sm" />
								<span class="component-name">{component.name}</span>
							</div>
						{/each}
					</div>
				</div>

				{#if $productivityScore.recommendations && $productivityScore.recommendations.length > 0}
					<div class="recommendations">
						<h3>{$t('dashboard.recommendations')}</h3>
						<ul>
							{#each $productivityScore.recommendations as rec}
								<li>{rec}</li>
							{/each}
						</ul>
					</div>
				{/if}
			</section>
		{/if}

		<!-- Key Metrics -->
		<section class="section">
			<h2 class="section-title">{$t('dashboard.keyMetrics')}</h2>
			<div class="grid grid-4">
				{#if $cycleTimeMetrics}
					<MetricCard
						title={$t('dashboard.avgCycleTime')}
						value={formatHours($cycleTimeMetrics.avgCycleTime)}
						subtitle={$t('dashboard.prToMerge')}
						color="primary"
					/>
					<MetricCard
						title={$t('dashboard.mergedPRs')}
						value={$cycleTimeMetrics.totalPRs}
						subtitle={$t('dashboard.inPeriod')}
						color="success"
					/>
				{/if}
				{#if $reviewMetrics}
					<MetricCard
						title={$t('dashboard.reviewCount')}
						value={$reviewMetrics.totalReviews}
						subtitle={$t('dashboard.totalReviews')}
						color="info"
					/>
					<MetricCard
						title={$t('dashboard.firstReviewTime')}
						value={formatHours($reviewMetrics.avgTimeToFirstReview)}
						subtitle={$t('dashboard.prToFirstReview')}
						color="warning"
					/>
				{/if}
			</div>
		</section>

		<!-- Cycle Time Chart -->
		{#if $dailyMetrics && $dailyMetrics.length > 0}
			<section class="section">
				<h2 class="section-title">{$t('dashboard.cycleTimeTrend')}</h2>
				<div class="card">
					<CycleTimeChart data={$dailyMetrics} />
				</div>
			</section>
		{/if}

		<!-- DORA Metrics -->
		{#if $doraMetrics}
			<section class="section">
				<h2 class="section-title">{$t('dashboard.doraMetrics')}</h2>
				<div class="grid grid-4">
					<MetricCard
						title={$t('dashboard.deployFrequency')}
						value={$t(`deployFrequency.${$doraMetrics.deploymentFrequency === 'daily' ? 'daily' : $doraMetrics.deploymentFrequency === 'weekly' ? 'weekly' : $doraMetrics.deploymentFrequency === 'monthly' ? 'monthly' : 'low'}`)}
						subtitle={$t('dashboard.deployCount', { count: $doraMetrics.deploymentCount })}
						color="primary"
					/>
					<MetricCard
						title={$t('dashboard.leadTime')}
						value={formatHours($doraMetrics.avgLeadTime)}
						subtitle={$t('dashboard.commitToProd')}
						color="info"
					/>
					<MetricCard
						title={$t('dashboard.changeFailureRate')}
						value="{$doraMetrics.changeFailureRate.toFixed(1)}%"
						subtitle="{$doraMetrics.failedChanges}/{$doraMetrics.totalChanges}"
						color={$doraMetrics.changeFailureRate > 15 ? "danger" : "success"}
					/>
					<MetricCard
						title={$t('dashboard.mttr')}
						value={$doraMetrics.avgMTTR > 0
              ? formatHours($doraMetrics.avgMTTR)
              : "N/A"}
						subtitle={$t('dashboard.avgRecoveryTime')}
						color="warning"
					/>
				</div>
			</section>
		{/if}

		<!-- PR Activity -->
		{#if $dailyMetrics && $dailyMetrics.length > 0}
			<section class="section">
				<h2 class="section-title">{$t('dashboard.prActivity')}</h2>
				<div class="card">
					<PRActivityChart data={$dailyMetrics} />
				</div>
			</section>
		{/if}

		<!-- Review Analysis -->
		{#if $reviewMetrics && $reviewMetrics.byReviewer && $reviewMetrics.byReviewer.length > 0}
			<section class="section">
				<h2 class="section-title">{$t('dashboard.reviewerAnalysis')}</h2>
				<div class="card">
					<table class="table">
						<thead>
							<tr>
								<th>{$t('dashboard.reviewer')}</th>
								<th>{$t('dashboard.reviewCountCol')}</th>
								<th>{$t('dashboard.commentCount')}</th>
								<th>{$t('dashboard.approvalRate')}</th>
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
			{@const sorted = [...filtered].sort(
        (a, b) => b.additions + b.deletions - (a.additions + a.deletions),
      )}
			{@const top10 = sorted.slice(0, 10)}
			{@const rest = sorted.slice(10)}
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
					<h2 class="section-title">{$t('dashboard.fileExtChanges')}</h2>
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
								<th>{$t('dashboard.extension')}</th>
								<th class="num">{$t('dashboard.addedLines')}</th>
								<th class="num">{$t('dashboard.deletedLines')}</th>
								<th class="num">{$t('dashboard.fileCount')}</th>
								<th class="num">{$t('dashboard.prCount')}</th>
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

.section-header {
	display: flex;
	align-items: center;
	gap: 1rem;
	margin-bottom: 1rem;
}

.section-title {
	font-size: 1.125rem;
	margin-bottom: 1rem;
	color: var(--color-text);
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

.page-header h1 {
	font-size: 1.75rem;
}

.refresh-btn {
	display: inline-flex;
	align-items: center;
	gap: 0.375rem;
	margin-left: auto;
	padding: 0.5rem 1rem;
	font-size: 0.875rem;
	color: var(--color-text);
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	cursor: pointer;
	transition: background 0.15s;
}

.refresh-btn:hover:not(:disabled) {
	background: var(--color-bg);
}

.refresh-btn:disabled {
	opacity: 0.5;
	cursor: not-allowed;
}

.refresh-icon {
	flex-shrink: 0;
}

.refresh-icon.spinning {
	animation: spin 1s linear infinite;
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

.table .num {
	text-align: right;
	font-variant-numeric: tabular-nums;
}

.table .additions {
	color: #22c55e;
}

.table .deletions {
	color: #ef4444;
}

.table .other-row {
	border-top: 2px solid var(--color-border);
	font-style: italic;
	color: var(--color-text-muted);
}

.table code {
	font-size: 0.875rem;
	padding: 0.125rem 0.375rem;
	background: var(--color-bg);
	border-radius: var(--radius-sm, 4px);
}

@media (max-width: 1024px) {
	.score-overview {
		flex-direction: column;
	}

	.score-components {
		grid-template-columns: repeat(2, 1fr);
	}
}
</style>
