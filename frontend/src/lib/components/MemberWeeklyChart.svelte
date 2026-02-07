<script lang="ts">
import type {
	Chart as ChartImport,
	Chart as ChartInstance,
	LegendItem,
	TooltipItem,
} from 'chart.js';
import { onDestroy, onMount } from 'svelte';
import { get } from 'svelte/store';
import type { MemberPullRequest, MemberReview } from '$api/client';
import { t } from '$i18n';

let {
	pullRequests,
	reviews = [],
	dateStart,
	dateEnd,
}: {
	pullRequests: MemberPullRequest[];
	reviews?: MemberReview[];
	dateStart: string;
	dateEnd: string;
} = $props();

let canvas: HTMLCanvasElement;
let chart: ChartInstance | null = null;
let ChartJS: typeof ChartImport | null = null;
let stackByRepo = $state(false);

const REPO_COLORS = [
	'#3b82f6',
	'#10b981',
	'#f59e0b',
	'#8b5cf6',
	'#ec4899',
	'#06b6d4',
	'#84cc16',
	'#f97316',
	'#6366f1',
	'#14b8a6',
];

interface WeeklyData {
	weekLabel: string;
	weekStart: string;
	prsCreated: number;
	prsMerged: number;
	reviewsGiven: number;
	additions: number;
	deletions: number;
}

interface WeeklyRepoData {
	weekLabel: string;
	weekStart: string;
	byRepo: Map<
		string,
		{
			prsCreated: number;
			prsMerged: number;
			reviewsGiven: number;
			additions: number;
			deletions: number;
		}
	>;
}

function getWeekMonday(date: Date): Date {
	const d = new Date(date);
	const day = d.getDay();
	const diff = day === 0 ? -6 : 1 - day;
	d.setDate(d.getDate() + diff);
	d.setHours(0, 0, 0, 0);
	return d;
}

function makeWeekLabel(d: Date): string {
	const weekEnd = new Date(d);
	weekEnd.setDate(weekEnd.getDate() + 6);
	return (
		d.toLocaleDateString('ja-JP', { month: 'short', day: 'numeric' }) +
		'~' +
		weekEnd.toLocaleDateString('ja-JP', { month: 'short', day: 'numeric' })
	);
}

function initWeekMap<T>(
	start: string,
	end: string,
	factory: (key: string, label: string) => T,
): Map<string, T> {
	const map = new Map<string, T>();
	const startDate = getWeekMonday(new Date(start));
	const endDate = new Date(end);
	for (let d = new Date(startDate); d <= endDate; d.setDate(d.getDate() + 7)) {
		const key = d.toISOString().slice(0, 10);
		map.set(key, factory(key, makeWeekLabel(d)));
	}
	return map;
}

// Normal mode: aggregated totals
function aggregateWeekly(
	prs: MemberPullRequest[],
	rvs: MemberReview[],
	start: string,
	end: string,
): WeeklyData[] {
	const map = initWeekMap<WeeklyData>(start, end, (key, label) => ({
		weekLabel: label,
		weekStart: key,
		prsCreated: 0,
		prsMerged: 0,
		reviewsGiven: 0,
		additions: 0,
		deletions: 0,
	}));
	for (const pr of prs) {
		const mondayKey = getWeekMonday(new Date(pr.createdAt)).toISOString().slice(0, 10);
		const entry = map.get(mondayKey);
		if (entry) {
			entry.prsCreated++;
			entry.additions += pr.additions;
			entry.deletions += pr.deletions;
		}
		if (pr.mergedAt) {
			const mKey = getWeekMonday(new Date(pr.mergedAt)).toISOString().slice(0, 10);
			const m = map.get(mKey);
			if (m) m.prsMerged++;
		}
	}
	for (const rv of rvs) {
		const mondayKey = getWeekMonday(new Date(rv.submittedAt)).toISOString().slice(0, 10);
		const entry = map.get(mondayKey);
		if (entry) entry.reviewsGiven++;
	}
	return Array.from(map.values()).sort((a, b) => a.weekStart.localeCompare(b.weekStart));
}

// Per-repository mode
function aggregateWeeklyByRepo(
	prs: MemberPullRequest[],
	rvs: MemberReview[],
	start: string,
	end: string,
) {
	const repoSet = new Set(prs.map((p) => p.repoName));
	rvs.forEach((rv) => repoSet.add(rv.repoName));
	const repos = [...repoSet].sort();
	const map = initWeekMap<WeeklyRepoData>(start, end, (key, label) => ({
		weekLabel: label,
		weekStart: key,
		byRepo: new Map(),
	}));
	const emptyEntry = () => ({
		prsCreated: 0,
		prsMerged: 0,
		reviewsGiven: 0,
		additions: 0,
		deletions: 0,
	});
	for (const pr of prs) {
		const mondayKey = getWeekMonday(new Date(pr.createdAt)).toISOString().slice(0, 10);
		const entry = map.get(mondayKey);
		if (entry) {
			if (!entry.byRepo.has(pr.repoName)) entry.byRepo.set(pr.repoName, emptyEntry());
			const r = entry.byRepo.get(pr.repoName)!;
			r.prsCreated++;
			r.additions += pr.additions;
			r.deletions += pr.deletions;
		}
		if (pr.mergedAt) {
			const mKey = getWeekMonday(new Date(pr.mergedAt)).toISOString().slice(0, 10);
			const m = map.get(mKey);
			if (m) {
				if (!m.byRepo.has(pr.repoName)) m.byRepo.set(pr.repoName, emptyEntry());
				m.byRepo.get(pr.repoName)!.prsMerged++;
			}
		}
	}
	for (const rv of rvs) {
		const mondayKey = getWeekMonday(new Date(rv.submittedAt)).toISOString().slice(0, 10);
		const entry = map.get(mondayKey);
		if (entry) {
			if (!entry.byRepo.has(rv.repoName)) entry.byRepo.set(rv.repoName, emptyEntry());
			entry.byRepo.get(rv.repoName)!.reviewsGiven++;
		}
	}
	const weeks = Array.from(map.values()).sort((a, b) => a.weekStart.localeCompare(b.weekStart));
	return { repos, weeks };
}

function rebuildChart() {
	if (!ChartJS || !canvas) return;
	if (chart) chart.destroy();

	if (!canvas) return;
	const ctx = canvas.getContext('2d');
	if (!ctx) return;

	const tr = get(t);

	if (stackByRepo) {
		const { repos, weeks } = aggregateWeeklyByRepo(pullRequests, reviews, dateStart, dateEnd);
		const labels = weeks.map((w) => w.weekLabel);

		// PRs created: stacked by repo
		const createdDatasets = repos.map((repo, i) => ({
			label: `${repo}`,
			data: weeks.map((w) => w.byRepo.get(repo)?.prsCreated ?? 0),
			backgroundColor: REPO_COLORS[i % REPO_COLORS.length],
			borderRadius: 2,
			stack: 'created',
			yAxisID: 'y',
		}));

		// Reviews: stacked by repo (right axis)
		const reviewDatasets = repos.map((repo, i) => ({
			label: `${repo} ${tr('chart.reviewLabel')}`,
			data: weeks.map((w) => w.byRepo.get(repo)?.reviewsGiven ?? 0),
			backgroundColor: `${REPO_COLORS[i % REPO_COLORS.length]}55`,
			borderColor: REPO_COLORS[i % REPO_COLORS.length],
			borderWidth: 1,
			borderRadius: 2,
			stack: 'reviews',
			yAxisID: 'y1',
		}));

		chart = new ChartJS(ctx, {
			type: 'bar',
			data: { labels, datasets: [...createdDatasets, ...reviewDatasets] },
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: {
						position: 'top',
						labels: {
							usePointStyle: true,
							padding: 12,
							font: { size: 10 },
							filter: (item: LegendItem) => !item.text?.includes(tr('chart.reviewLabel')),
						},
					},
					tooltip: {
						callbacks: {
							afterBody: (items: TooltipItem<'bar'>[]) => {
								const idx = items[0]?.dataIndex;
								if (idx == null) return '';
								const w = weeks[idx];
								let total = 0,
									add = 0,
									del = 0,
									rv = 0;
								w.byRepo.forEach((v) => {
									total += v.prsCreated;
									add += v.additions;
									del += v.deletions;
									rv += v.reviewsGiven;
								});
								const tr2 = get(t);
								return (
									tr2('chart.totalPR', { count: String(total), reviews: String(rv) }) +
									'\n' +
									tr2('chart.totalLines', { add: add.toLocaleString(), del: del.toLocaleString() })
								);
							},
						},
					},
				},
				scales: {
					x: { stacked: true, ticks: { maxRotation: 45, autoSkipPadding: 8, font: { size: 10 } } },
					y: {
						stacked: true,
						position: 'left',
						beginAtZero: true,
						title: { display: true, text: tr('chart.prsCreatedCount') },
						ticks: { stepSize: 1 },
					},
					y1: {
						stacked: true,
						position: 'right',
						beginAtZero: true,
						title: { display: true, text: tr('chart.reviewCount') },
						grid: { drawOnChartArea: false },
						ticks: { stepSize: 1 },
					},
				},
			},
		});
	} else {
		const data = aggregateWeekly(pullRequests, reviews, dateStart, dateEnd);
		chart = new ChartJS(ctx, {
			type: 'bar',
			data: {
				labels: data.map((d) => d.weekLabel),
				datasets: [
					{
						label: tr('chart.prsCreated'),
						data: data.map((d) => d.prsCreated),
						backgroundColor: '#3b82f6',
						borderRadius: 4,
						yAxisID: 'y',
					},
					{
						label: tr('chart.prsMerged'),
						data: data.map((d) => d.prsMerged),
						backgroundColor: '#10b981',
						borderRadius: 4,
						yAxisID: 'y',
					},
					{
						label: tr('chart.reviewLabel'),
						data: data.map((d) => d.reviewsGiven),
						backgroundColor: '#8b5cf6',
						borderRadius: 4,
						yAxisID: 'y',
					},
					{
						label: tr('chart.addedLines'),
						data: data.map((d) => d.additions),
						backgroundColor: 'rgba(34, 197, 94, 0.3)',
						borderColor: '#22c55e',
						borderWidth: 1,
						borderRadius: 4,
						yAxisID: 'y1',
					},
					{
						label: tr('chart.deletedLines'),
						data: data.map((d) => d.deletions),
						backgroundColor: 'rgba(239, 68, 68, 0.3)',
						borderColor: '#ef4444',
						borderWidth: 1,
						borderRadius: 4,
						yAxisID: 'y1',
					},
				],
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: { position: 'top', labels: { usePointStyle: true, padding: 16 } },
				},
				scales: {
					x: { ticks: { maxRotation: 45, autoSkipPadding: 8, font: { size: 10 } } },
					y: {
						type: 'linear',
						position: 'left',
						beginAtZero: true,
						title: { display: true, text: tr('chart.count') },
						ticks: { stepSize: 1 },
					},
					y1: {
						type: 'linear',
						position: 'right',
						beginAtZero: true,
						title: { display: true, text: tr('chart.codeLines') },
						grid: { drawOnChartArea: false },
					},
				},
			},
		});
	}
}

onMount(async () => {
	const mod = await import('chart.js');
	ChartJS = mod.Chart;
	mod.Chart.register(...mod.registerables);
	rebuildChart();
});

onDestroy(() => {
	if (chart) chart.destroy();
});

$effect(() => {
	const _byRepo = stackByRepo;
	if (ChartJS && pullRequests && reviews) rebuildChart();
});
</script>

<div class="chart-wrapper">
	<div class="chart-toolbar">
		<label class="toggle-label">
			<span class="toggle-text">{$t('chart.byRepo')}</span>
			<button
				class="toggle-switch"
				class:active={stackByRepo}
				onclick={() => { stackByRepo = !stackByRepo; }}
				role="switch"
				aria-checked={stackByRepo}
				aria-label={$t('chart.byRepo')}
			>
				<span class="toggle-knob"></span>
			</button>
		</label>
	</div>
	<div class="chart-container">
		<canvas bind:this={canvas}></canvas>
	</div>
</div>

<style>
.chart-wrapper {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.chart-toolbar {
	display: flex;
	justify-content: flex-end;
	padding: 0 0.25rem;
}

.toggle-label {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	cursor: pointer;
	user-select: none;
}

.toggle-text {
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.toggle-switch {
	position: relative;
	width: 36px;
	height: 20px;
	border-radius: 10px;
	border: none;
	background: var(--color-border, #d1d5db);
	cursor: pointer;
	transition: background 0.2s;
	padding: 0;
}

.toggle-switch.active {
	background: var(--color-primary, #3b82f6);
}

.toggle-knob {
	position: absolute;
	top: 2px;
	left: 2px;
	width: 16px;
	height: 16px;
	border-radius: 50%;
	background: white;
	transition: transform 0.2s;
	box-shadow: 0 1px 3px rgba(0, 0, 0, 0.15);
}

.toggle-switch.active .toggle-knob {
	transform: translateX(16px);
}

.chart-container {
	position: relative;
	height: 300px;
	width: 100%;
}
</style>
