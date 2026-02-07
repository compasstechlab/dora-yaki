import { writable } from 'svelte/store';
import {
	api,
	type CycleTimeMetrics,
	type DailyMetrics,
	type DORAMetrics,
	type ProductivityScore,
	type ReviewMetrics,
} from '$api/client';

// 期間プリセット
export type PeriodPreset = 'week' | '2week' | 'month' | '3month' | '6month' | 'year';

export const PERIOD_OPTIONS: { key: PeriodPreset; days: number; labelKey: string }[] = [
	{ key: 'week', days: 7, labelKey: 'period.week' },
	{ key: '2week', days: 14, labelKey: 'period.2week' },
	{ key: 'month', days: 30, labelKey: 'period.month' },
	{ key: '3month', days: 90, labelKey: 'period.3month' },
	{ key: '6month', days: 180, labelKey: 'period.6month' },
	{ key: 'year', days: 365, labelKey: 'period.year' },
];

export const selectedPeriod = writable<PeriodPreset>('month');

// Date range store
export const dateRange = writable<{ start: string; end: string }>({
	start: getDefaultStartDate(),
	end: getDefaultEndDate(),
});

// Convert a Date to YYYY-MM-DD string.
// Date を YYYY-MM-DD 形式の文字列に変換する
function toDateString(date: Date): string {
	return date.toISOString().split('T')[0];
}

function getStartDateForPeriod(period: PeriodPreset): string {
	const opt = PERIOD_OPTIONS.find((o) => o.key === period);
	const days = opt?.days ?? 30;
	const date = new Date();
	date.setDate(date.getDate() - days);
	return toDateString(date);
}

function getDefaultStartDate(): string {
	const date = new Date();
	date.setMonth(date.getMonth() - 1);
	return toDateString(date);
}

function getDefaultEndDate(): string {
	return toDateString(new Date());
}

// 期間プリセットを変更し、dateRangeを再計算
export function setPeriod(period: PeriodPreset) {
	selectedPeriod.set(period);
	dateRange.set({
		start: getStartDateForPeriod(period),
		end: getDefaultEndDate(),
	});
}

// Metrics stores
export const cycleTimeMetrics = writable<CycleTimeMetrics | null>(null);
export const reviewMetrics = writable<ReviewMetrics | null>(null);
export const doraMetrics = writable<DORAMetrics | null>(null);
export const productivityScore = writable<ProductivityScore | null>(null);
export const dailyMetrics = writable<DailyMetrics[]>([]);

// Loading state
export const isLoading = writable(false);

// Load metrics for multiple repositories
// Empty array or undefined repositoryIds = all repos (no filter param)
// refresh: true bypasses server cache
export async function loadMetrics(
	repositoryIds?: string[],
	start?: string,
	end?: string,
	refresh?: boolean,
) {
	isLoading.set(true);

	// Convert empty array to undefined (means all repos)
	const repos = repositoryIds && repositoryIds.length > 0 ? repositoryIds : undefined;

	try {
		const [cycleTime, reviews, dora, score, daily] = await Promise.all([
			api.metrics.cycleTime(repos, start, end, refresh),
			api.metrics.reviews(repos, start, end, refresh),
			api.metrics.dora(repos, start, end, refresh),
			api.metrics.productivityScore(repos, start, end, refresh),
			api.metrics.daily(repos, start, end, refresh),
		]);

		cycleTimeMetrics.set(cycleTime);
		reviewMetrics.set(reviews);
		doraMetrics.set(dora);
		productivityScore.set(score);
		dailyMetrics.set(daily);
	} catch (error) {
		console.error('Failed to load metrics:', error);
		throw error;
	} finally {
		isLoading.set(false);
	}
}

// Clear all metrics
export function clearMetrics() {
	cycleTimeMetrics.set(null);
	reviewMetrics.set(null);
	doraMetrics.set(null);
	productivityScore.set(null);
	dailyMetrics.set([]);
}

// Utility functions
export function formatHours(hours: number): string {
	if (hours < 1) {
		return `${Math.round(hours * 60)}m`;
	}
	if (hours < 24) {
		return `${hours.toFixed(1)}h`;
	}
	return `${(hours / 24).toFixed(1)}d`;
}

export function getScoreClass(score: number): string {
	if (score >= 80) return 'score-excellent';
	if (score >= 60) return 'score-good';
	if (score >= 40) return 'score-average';
	return 'score-poor';
}

export function getTrendClass(direction: string): string {
	switch (direction) {
		case 'up':
			return 'trend-up';
		case 'down':
			return 'trend-down';
		default:
			return 'trend-stable';
	}
}
