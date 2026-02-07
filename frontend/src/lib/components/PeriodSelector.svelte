<script lang="ts">
import { t } from '$i18n';
import { PERIOD_OPTIONS, selectedPeriod, setPeriod } from '$stores/metrics';
</script>

<div class="period-selector">
	<span class="period-label">{$t('period.label')}</span>
	<div class="period-buttons">
		{#each PERIOD_OPTIONS as opt}
			<button
				class="period-btn"
				class:active={$selectedPeriod === opt.key}
				onclick={() => setPeriod(opt.key)}
			>
				{$t(opt.labelKey)}
			</button>
		{/each}
	</div>
</div>

<style>
.period-selector {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
	margin-bottom: 0.75rem;
	padding-bottom: 0.75rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.period-label {
	font-size: 0.75rem;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	color: rgba(255, 255, 255, 0.5);
}

.period-buttons {
	display: grid;
	grid-template-columns: repeat(4, 1fr);
	gap: 0.25rem;
}

.period-btn {
	padding: 0.375rem 0;
	font-size: 0.6875rem;
	font-weight: 500;
	color: rgba(255, 255, 255, 0.6);
	background: rgba(255, 255, 255, 0.06);
	border: 1px solid rgba(255, 255, 255, 0.12);
	border-radius: var(--radius-sm);
	cursor: pointer;
	transition: all 0.15s ease;
	white-space: nowrap;
}

.period-btn:hover {
	background: rgba(255, 255, 255, 0.12);
	color: rgba(255, 255, 255, 0.9);
}

.period-btn.active {
	background: rgba(59, 130, 246, 0.3);
	border-color: rgba(59, 130, 246, 0.5);
	color: white;
	font-weight: 600;
}
</style>
