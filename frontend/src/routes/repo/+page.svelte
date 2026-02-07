<script lang="ts">
import { onMount } from 'svelte';
import {
	api,
	type CycleTimeMetrics,
	type DORAMetrics,
	type ProductivityScore,
	type Repository,
	type ReviewMetrics,
} from '$api/client';
import { browser } from '$app/environment';
import ScoreGauge from '$components/ScoreGauge.svelte';
import { t } from '$i18n';
import { dateRange, formatHours } from '$stores/metrics';
import { repositories } from '$stores/repositories';

interface RepoSummary {
	repo: Repository;
	cycleTime: CycleTimeMetrics | null;
	reviews: ReviewMetrics | null;
	dora: DORAMetrics | null;
	score: ProductivityScore | null;
}

let summaries: RepoSummary[] = $state([]);
let isLoading = $state(true);
let reposLoaded = $state(false);

onMount(() => {
	reposLoaded = true;
});

$effect(() => {
	if (browser && reposLoaded && $repositories.length > 0 && $dateRange) {
		loadSummaries();
	}
});

async function loadSummaries() {
	isLoading = true;
	try {
		summaries = await Promise.all(
			$repositories.map(async (repo) => {
				try {
					const [cycleTime, reviews, dora, score] = await Promise.all([
						api.metrics.cycleTime([repo.id], $dateRange.start, $dateRange.end),
						api.metrics.reviews([repo.id], $dateRange.start, $dateRange.end),
						api.metrics.dora([repo.id], $dateRange.start, $dateRange.end),
						api.metrics.productivityScore([repo.id], $dateRange.start, $dateRange.end),
					]);
					return { repo, cycleTime, reviews, dora, score };
				} catch {
					return { repo, cycleTime: null, reviews: null, dora: null, score: null };
				}
			}),
		);
	} finally {
		isLoading = false;
	}
}
</script>

<svelte:head>
	<title>{$t('repo.pageTitle')}</title>
</svelte:head>

<div class="page">
	<header class="page-header">
		<h1>{$t('repo.title')}</h1>
		<span class="repo-count">{$t('repo.repoCount', { count: $repositories.length })}</span>
	</header>

	{#if isLoading}
		<div class="loading">
			<div class="spinner"></div>
			<p>{$t('common.loadingMetrics')}</p>
		</div>
	{:else if summaries.length === 0}
		<div class="empty">
			<p>{$t('repo.noRepos')}</p>
			<a href="/repositories" class="link">{$t('repo.addRepoLink')}</a>
		</div>
	{:else}
		<div class="repo-grid">
			{#each summaries as { repo, cycleTime, reviews, dora, score }}
				<a href="/repo/{repo.id}" class="repo-card">
					<div class="repo-card-header">
						<h2 class="repo-name">{repo.fullName}</h2>
						{#if score}
							<div class="repo-score">
								<ScoreGauge score={score.overallScore} label="" size="sm" />
							</div>
						{/if}
					</div>

					<div class="repo-metrics">
						<div class="metric-item">
							<span class="metric-label">{$t('repo.avgCycleTime')}</span>
							<span class="metric-value">
								{cycleTime ? formatHours(cycleTime.avgCycleTime) : 'N/A'}
							</span>
						</div>
						<div class="metric-item">
							<span class="metric-label">{$t('repo.mergedPRs')}</span>
							<span class="metric-value"> {cycleTime ? cycleTime.totalPRs : 'N/A'} </span>
						</div>
						<div class="metric-item">
							<span class="metric-label">{$t('repo.reviewCount')}</span>
							<span class="metric-value"> {reviews ? reviews.totalReviews : 'N/A'} </span>
						</div>
						<div class="metric-item">
							<span class="metric-label">{$t('repo.deployFrequency')}</span>
							<span class="metric-value">
								{dora ? $t(`deployFrequency.${dora.deploymentFrequency === 'daily' ? 'daily' : dora.deploymentFrequency === 'weekly' ? 'weekly' : dora.deploymentFrequency === 'monthly' ? 'monthly' : 'low'}`) : 'N/A'}
							</span>
						</div>
					</div>
				</a>
			{/each}
		</div>
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

.repo-count {
	font-size: 0.875rem;
	padding: 0.25rem 0.75rem;
	background: var(--color-primary);
	color: white;
	border-radius: var(--radius-md);
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

.empty {
	text-align: center;
	padding: 4rem;
	color: var(--color-text-muted);
}

.empty .link {
	color: var(--color-primary);
	text-decoration: underline;
}

.repo-grid {
	display: grid;
	grid-template-columns: repeat(2, 1fr);
	gap: 1.5rem;
}

.repo-card {
	display: block;
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	padding: 1.5rem;
	text-decoration: none;
	color: inherit;
	transition:
		border-color 0.15s ease,
		box-shadow 0.15s ease;
}

.repo-card:hover {
	border-color: var(--color-primary);
	box-shadow: 0 2px 8px rgba(59, 130, 246, 0.15);
	text-decoration: none;
}

.repo-card-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-bottom: 1.25rem;
}

.repo-name {
	font-size: 1rem;
	font-weight: 600;
	color: var(--color-text);
}

.repo-score {
	flex-shrink: 0;
}

.repo-metrics {
	display: grid;
	grid-template-columns: repeat(2, 1fr);
	gap: 1rem;
}

.metric-item {
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
}

.metric-label {
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.metric-value {
	font-size: 1.125rem;
	font-weight: 600;
}

@media (max-width: 768px) {
	.repo-grid {
		grid-template-columns: 1fr;
	}
}
</style>
