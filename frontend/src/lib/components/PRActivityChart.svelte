<script lang="ts">
import type { Chart as ChartInstance } from 'chart.js';
import { onDestroy, onMount } from 'svelte';
import { get } from 'svelte/store';
import type { DailyMetrics } from '$api/client';
import { t } from '$i18n';

let { data }: { data: DailyMetrics[] } = $props();

let canvas: HTMLCanvasElement;
let chart: ChartInstance | null = null;

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
		type: 'bar',
		data: {
			labels,
			datasets: [
				{
					label: tr('chart.prOpen'),
					data: data.map((d) => d.prsOpened),
					backgroundColor: '#3b82f6',
					borderRadius: 4,
				},
				{
					label: tr('chart.prMerge'),
					data: data.map((d) => d.prsMerged),
					backgroundColor: '#10b981',
					borderRadius: 4,
				},
				{
					label: tr('chart.reviewLabel'),
					data: data.map((d) => d.reviewsSubmitted),
					backgroundColor: '#8b5cf6',
					borderRadius: 4,
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
			},
			scales: {
				x: {
					stacked: false,
				},
				y: {
					beginAtZero: true,
					title: {
						display: true,
						text: tr('chart.count'),
					},
				},
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
		chart.data.datasets[0].label = tr('chart.prOpen');
		chart.data.datasets[0].data = data.map((d) => d.prsOpened);
		chart.data.datasets[1].label = tr('chart.prMerge');
		chart.data.datasets[1].data = data.map((d) => d.prsMerged);
		chart.data.datasets[2].label = tr('chart.reviewLabel');
		chart.data.datasets[2].data = data.map((d) => d.reviewsSubmitted);
		chart.options.scales.y.title.text = tr('chart.count');
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
