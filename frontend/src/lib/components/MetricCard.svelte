<script lang="ts">
interface Props {
	title: string;
	value: string | number;
	subtitle?: string;
	trend?: 'up' | 'down' | 'stable' | null;
	trendValue?: string;
	color?: 'primary' | 'success' | 'warning' | 'danger' | 'info';
}

let {
	title,
	value,
	subtitle = '',
	trend = null,
	trendValue = '',
	color = 'primary',
}: Props = $props();

function getTrendIcon(t: 'up' | 'down' | 'stable' | null): string {
	switch (t) {
		case 'up':
			return '↑';
		case 'down':
			return '↓';
		default:
			return '→';
	}
}
</script>

<div class="metric-card">
	<div class="metric-header">
		<span class="metric-title">{title}</span>
		{#if trend}
			<span class="trend trend-{trend}"> {getTrendIcon(trend)} {trendValue} </span>
		{/if}
	</div>
	<div class="metric-value color-{color}">{value}</div>
	{#if subtitle}
		<div class="metric-subtitle">{subtitle}</div>
	{/if}
</div>

<style>
.metric-card {
	background: var(--color-bg-card);
	border-radius: var(--radius-lg);
	padding: 1.25rem;
	border: 1px solid var(--color-border);
}

.metric-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 0.5rem;
}

.metric-title {
	font-size: 0.875rem;
	color: var(--color-text-muted);
	font-weight: 500;
}

.metric-value {
	font-size: 2rem;
	font-weight: 700;
	line-height: 1.2;
}

.metric-subtitle {
	font-size: 0.75rem;
	color: var(--color-text-muted);
	margin-top: 0.25rem;
}

.trend {
	font-size: 0.75rem;
	font-weight: 500;
	padding: 0.125rem 0.5rem;
	border-radius: var(--radius-sm);
}

.trend-up {
	color: var(--color-success);
	background: rgba(16, 185, 129, 0.1);
}

.trend-down {
	color: var(--color-danger);
	background: rgba(239, 68, 68, 0.1);
}

.trend-stable {
	color: var(--color-text-muted);
	background: rgba(100, 116, 139, 0.1);
}

.color-primary {
	color: var(--color-primary);
}

.color-success {
	color: var(--color-success);
}

.color-warning {
	color: var(--color-warning);
}

.color-danger {
	color: var(--color-danger);
}

.color-info {
	color: var(--color-info);
}
</style>
