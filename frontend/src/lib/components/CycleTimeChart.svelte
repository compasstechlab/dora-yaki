<script lang="ts">
import type { Chart as ChartInstance } from 'chart.js';
import { onDestroy, onMount } from 'svelte';
import { get } from 'svelte/store';
import type { DailyMetrics } from '$api/client';
import { t } from '$i18n';

let { data }: { data: DailyMetrics[] } = $props();

let canvas: HTMLCanvasElement;
let chart: ChartInstance | null = null;

function formatHours(hours: number): string {
	const tr = get(t);
	if (hours < 1) return `${Math.round(hours * 60)}${tr('chart.minutes')}`;
	if (hours < 24) return `${hours.toFixed(1)}${tr('chart.hoursUnit')}`;
	return `${(hours / 24).toFixed(1)}${tr('chart.days')}`;
}

onMount(async () => {
	const { Chart, registerables } = await import('chart.js');
	Chart.register(...registerables);

	if (!canvas) return;
	const ctx = canvas.getContext('2d');
	if (!ctx) return;

	const tr = get(t);

	const labels = data.map((d) => {
		const date = new Date(d.date);
		return date.toLocaleDateString('ja-JP', { month: 'short', day: 'numeric' });
	});

	chart = new Chart(ctx, {
		type: 'line',
		data: {
			labels,
			datasets: [
				{
					label: tr('chart.cycleTime'),
					data: data.map((d) => d.avgCycleTime),
					borderColor: '#3b82f6',
					backgroundColor: 'rgba(59, 130, 246, 0.1)',
					fill: true,
					tension: 0.4,
				},
				{
					label: tr('chart.pickupTime'),
					data: data.map((d) => d.avgPickupTime),
					borderColor: '#8b5cf6',
					backgroundColor: 'transparent',
					tension: 0.4,
				},
				{
					label: tr('chart.reviewTime'),
					data: data.map((d) => d.avgReviewTime),
					borderColor: '#06b6d4',
					backgroundColor: 'transparent',
					tension: 0.4,
				},
			],
		},
		options: {
			responsive: true,
			maintainAspectRatio: false,
			plugins: {
				legend: {
					position: 'top',
					labels: {
						usePointStyle: true,
						padding: 20,
					},
				},
				tooltip: {
					mode: 'index',
					intersect: false,
					callbacks: {
						label: (context) => {
							const value = context.parsed.y ?? 0;
							return `${context.dataset.label}: ${formatHours(value)}`;
						},
					},
				},
			},
			scales: {
				y: {
					beginAtZero: true,
					title: {
						display: true,
						text: tr('chart.hours'),
					},
				},
			},
			interaction: {
				mode: 'nearest',
				axis: 'x',
				intersect: false,
			},
		},
	});
});

onDestroy(() => {
	if (chart) {
		chart.destroy();
	}
});

$effect(() => {
	if (chart && data) {
		const tr = get(t);
		chart.data.labels = data.map((d) => {
			const date = new Date(d.date);
			return date.toLocaleDateString('ja-JP', { month: 'short', day: 'numeric' });
		});
		chart.data.datasets[0].label = tr('chart.cycleTime');
		chart.data.datasets[0].data = data.map((d) => d.avgCycleTime);
		chart.data.datasets[1].label = tr('chart.pickupTime');
		chart.data.datasets[1].data = data.map((d) => d.avgPickupTime);
		chart.data.datasets[2].label = tr('chart.reviewTime');
		chart.data.datasets[2].data = data.map((d) => d.avgReviewTime);
		chart.options.scales.y.title.text = tr('chart.hours');
		chart.update();
	}
});
</script>

<div class="chart-container">
	<canvas bind:this={canvas}></canvas>
</div>

<style>
.chart-container {
	position: relative;
	height: 300px;
	width: 100%;
}
</style>
