<script lang="ts">
interface Props {
	score: number;
	label: string;
	size?: 'sm' | 'md' | 'lg';
}

let { score, label, size = 'md' }: Props = $props();

let percentage = $derived(Math.min(Math.max(score, 0), 100));
let circumference = $derived(2 * Math.PI * 45);
let dashOffset = $derived(circumference - (percentage / 100) * circumference);
let scoreClass = $derived(getScoreClass(score));

function getScoreClass(s: number): string {
	if (s >= 80) return 'excellent';
	if (s >= 60) return 'good';
	if (s >= 40) return 'average';
	return 'poor';
}

const sizes = {
	sm: { width: 80, fontSize: '1rem' },
	md: { width: 120, fontSize: '1.5rem' },
	lg: { width: 160, fontSize: '2rem' },
};
</script>

<div class="score-gauge size-{size}">
	<svg viewBox="0 0 100 100" width={sizes[size].width} height={sizes[size].width}>
		<circle class="track" cx="50" cy="50" r="45" />
		<circle
			class="progress {scoreClass}"
			cx="50"
			cy="50"
			r="45"
			stroke-dasharray={circumference}
			stroke-dashoffset={dashOffset}
		/>
	</svg>
	<div class="score-content">
		<span class="score-value {scoreClass}" style="font-size: {sizes[size].fontSize}">
			{Math.round(score)}
		</span>
		{#if label}
			<span class="score-label">{label}</span>
		{/if}
	</div>
</div>

<style>
.score-gauge {
	position: relative;
	display: inline-flex;
	align-items: center;
	justify-content: center;
}

svg {
	transform: rotate(-90deg);
}

.track {
	fill: none;
	stroke: var(--color-border);
	stroke-width: 8;
}

.progress {
	fill: none;
	stroke-width: 8;
	stroke-linecap: round;
	transition: stroke-dashoffset 0.5s ease;
}

.progress.excellent {
	stroke: var(--color-success);
}

.progress.good {
	stroke: var(--color-info);
}

.progress.average {
	stroke: var(--color-warning);
}

.progress.poor {
	stroke: var(--color-danger);
}

.score-content {
	position: absolute;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
}

.score-value {
	font-weight: 700;
	line-height: 1;
}

.score-value.excellent {
	color: var(--color-success);
}

.score-value.good {
	color: var(--color-info);
}

.score-value.average {
	color: var(--color-warning);
}

.score-value.poor {
	color: var(--color-danger);
}

.score-label {
	font-size: 0.625rem;
	color: var(--color-text-muted);
	text-transform: uppercase;
	letter-spacing: 0.05em;
	margin-top: 0.25rem;
}

.size-sm .score-label {
	display: none;
}
</style>
