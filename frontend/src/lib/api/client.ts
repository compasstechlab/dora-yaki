import { addFlash } from '$stores/flash';

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

interface RequestOptions {
	method?: string;
	body?: unknown;
	headers?: Record<string, string>;
	/** If true, suppress flash messages on error */
	silent?: boolean;
}

async function request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
	const { method = 'GET', body, headers = {}, silent = false } = options;

	const config: RequestInit = {
		method,
		headers: {
			'Content-Type': 'application/json',
			...headers,
		},
	};

	if (body) {
		config.body = JSON.stringify(body);
	}

	let response: Response;
	try {
		response = await fetch(`${API_BASE}${endpoint}`, config);
	} catch (e) {
		const msg = e instanceof Error ? e.message : 'ネットワークエラーが発生しました';
		if (!silent) addFlash('error', `API通信エラー: ${msg}`);
		throw e;
	}

	if (!response.ok) {
		const error = await response.text();
		const msg = error || `HTTPエラー: ${response.status}`;
		if (!silent) addFlash('error', `APIエラー (${response.status}): ${msg}`);
		throw new Error(msg);
	}

	// Skip JSON parsing for empty responses (e.g. 204 No Content)
	if (response.status === 204 || response.headers.get('content-length') === '0') {
		return undefined as T;
	}

	return response.json();
}

// Types
export interface Repository {
	id: string;
	owner: string;
	name: string;
	fullName: string;
	private: boolean;
	createdAt: string;
	updatedAt: string;
	lastSyncedAt?: string;
}

export interface FileExtensionMetrics {
	extension: string;
	additions: number;
	deletions: number;
	files: number;
	prCount: number;
}

export interface CycleTimeMetrics {
	period: string;
	startDate: string;
	endDate: string;
	totalPRs: number;
	avgCycleTime: number;
	avgCodingTime: number;
	avgPickupTime: number;
	avgReviewTime: number;
	avgMergeTime: number;
	medianCycleTime: number;
	p90CycleTime: number;
	byAuthor?: AuthorMetrics[];
	dailyBreakdown?: DailyMetrics[];
	byFileExtension?: FileExtensionMetrics[];
}

export interface AuthorMetrics {
	author: string;
	prCount: number;
	avgCycleTime: number;
	additions: number;
	deletions: number;
}

export interface DailyMetrics {
	id: string;
	repositoryId: string;
	date: string;
	avgCycleTime: number;
	avgCodingTime: number;
	avgPickupTime: number;
	avgReviewTime: number;
	avgMergeTime: number;
	prsOpened: number;
	prsMerged: number;
	prsClosed: number;
	reviewsSubmitted: number;
	avgReviewsPerPR: number;
	totalAdditions: number;
	totalDeletions: number;
	deploymentCount: number;
	activeContributors: number;
}

export interface ReviewMetrics {
	period: string;
	startDate: string;
	endDate: string;
	totalReviews: number;
	totalComments: number;
	avgReviewsPerPR: number;
	avgCommentsPerReview: number;
	avgTimeToFirstReview: number;
	approvalRate: number;
	changesRequestedRate: number;
	byReviewer?: ReviewerStats[];
}

export interface ReviewerStats {
	reviewer: string;
	reviewCount: number;
	commentCount: number;
	avgResponseTime: number;
	approvalRate: number;
}

export interface DORAMetrics {
	period: string;
	startDate: string;
	endDate: string;
	deploymentCount: number;
	deploymentFrequency: string;
	avgDeploysPerDay: number;
	avgLeadTime: number;
	medianLeadTime: number;
	p90LeadTime: number;
	totalChanges: number;
	failedChanges: number;
	changeFailureRate: number;
	incidentCount: number;
	avgMTTR: number;
	medianMTTR: number;
}

export interface ProductivityScore {
	repositoryId: string;
	period: string;
	overallScore: number;
	cycleTimeScore: number;
	reviewScore: number;
	deploymentScore: number;
	qualityScore: number;
	trendDirection: string;
	trendPercentage: number;
	recommendations?: string[];
	componentScores?: ComponentScore[];
}

export interface ComponentScore {
	name: string;
	score: number;
	weight: number;
	description: string;
}

export interface Sprint {
	id: string;
	repositoryId: string;
	name: string;
	startDate: string;
	endDate: string;
	goals: string;
}

export interface SprintPerformance {
	sprintId: string;
	sprintName: string;
	startDate: string;
	endDate: string;
	status: string;
	plannedItems: number;
	completedItems: number;
	completionRate: number;
	prsOpened: number;
	prsMerged: number;
	avgPRSize: number;
	avgCycleTime: number;
	avgReviewTime: number;
	activeContributors: number;
	reviewsSubmitted: number;
	velocityChange: number;
	cycleTimeChange: number;
	burndownData?: BurndownPoint[];
}

export interface BurndownPoint {
	date: string;
	planned: number;
	remaining: number;
	completed: number;
}

export interface TeamMember {
	id: string;
	login: string;
	name: string;
	avatarUrl: string;
}

export interface MemberStats {
	member: TeamMember;
	prsAuthored: number;
	prsMerged: number;
	reviewsGiven: number;
	commentsGiven: number;
	avgCycleTime: number;
	avgCodingTime: number;
	avgPickupTime: number;
	avgReviewTime: number;
	avgMergeTime: number;
	reviewsApproved: number;
	reviewsChangesRequested: number;
	reviewsCommented: number;
	approvalRate: number;
	totalAdditions: number;
	totalDeletions: number;
	byFileExtension?: FileExtensionMetrics[];
}

export interface MemberPullRequest {
	number: number;
	title: string;
	author?: string;
	state: string;
	createdAt: string;
	mergedAt?: string;
	additions: number;
	deletions: number;
	cycleTime: number;
	codingTime: number;
	pickupTime: number;
	reviewTime: number;
	mergeTime: number;
	repoName: string;
}

export interface MemberReview {
	submittedAt: string;
	state: string;
	repoName: string;
}

export interface SyncResponse {
	repository: Repository;
	pullRequests: number;
	reviews: number;
	deployments: number;
	teamMembers: number;
	syncedAt: string;
}

export interface GitHubMe {
	login: string;
	name: string;
	avatarUrl: string;
	orgs: string[];
}

export interface GitHubOrgRepo {
	id: number;
	name: string;
	fullName: string;
	owner: string;
	private: boolean;
	description: string;
	language: string;
	archived: boolean;
}

export interface BatchAddResult {
	owner: string;
	name: string;
	success: boolean;
	error?: string;
	repository?: Repository;
}

export interface DataDateRange {
	repositoryId: string;
	oldestDate?: string;
	newestDate?: string;
	prCount: number;
}

export interface BotUser {
	username: string;
	createdAt: string;
}

/** Bot filtering options */
export interface BotFilterOptions {
	excludeBots?: boolean;
	botsOnly?: boolean;
}

// Build common URL params for metrics endpoints.
// メトリクス系エンドポイントの共通パラメータを構築する
function buildMetricsParams(
	repositories?: string[],
	start?: string,
	end?: string,
	refresh?: boolean,
	botFilter?: BotFilterOptions,
): URLSearchParams {
	const params = new URLSearchParams();
	if (repositories) {
		repositories.forEach((r) => params.append('repository', r));
	}
	if (start) params.append('start', start);
	if (end) params.append('end', end);
	if (refresh) params.append('refresh', 'true');
	if (botFilter?.excludeBots === false) params.append('exclude_bots', 'false');
	if (botFilter?.botsOnly) params.append('bots_only', 'true');
	return params;
}

// Build common URL params with date range and repositories.
// 日付範囲・リポジトリの共通パラメータを構築する
function buildDateRangeParams(
	repositories?: string[],
	start?: string,
	end?: string,
): URLSearchParams {
	const params = new URLSearchParams();
	if (repositories) {
		repositories.forEach((r) => params.append('repository', r));
	}
	if (start) params.append('start', start);
	if (end) params.append('end', end);
	return params;
}

// API functions
export const api = {
	// Cache
	cache: {
		invalidate: () => request<{ status: string }>('/cache/invalidate', { method: 'POST' }),
	},
	// Repositories
	repositories: {
		list: () => request<Repository[]>('/repositories'),
		add: (owner: string, name: string) =>
			request<Repository>('/repositories', { method: 'POST', body: { owner, name } }),
		get: (id: string) => request<Repository>(`/repositories/${id}`),
		delete: (id: string) => request<void>(`/repositories/${id}`, { method: 'DELETE' }),
		sync: (id: string, range?: string) => {
			const params = new URLSearchParams();
			if (range) params.append('range', range);
			return request<SyncResponse>(`/repositories/${id}/sync?${params}`, { method: 'POST' });
		},
		batchAdd: (repositories: { owner: string; name: string }[]) =>
			request<BatchAddResult[]>('/repositories/batch', { method: 'POST', body: { repositories } }),
		dateRanges: () => request<DataDateRange[]>('/repositories/date-ranges'),
	},

	// GitHub
	github: {
		getMe: () => request<GitHubMe>('/github/me'),
		listOwnerRepos: (owner: string, type?: string) => {
			const params = new URLSearchParams();
			if (type) params.append('type', type);
			return request<GitHubOrgRepo[]>(`/github/owners/${owner}/repos?${params}`);
		},
	},

	// Metrics
	metrics: {
		cycleTime: (
			repositories?: string[],
			start?: string,
			end?: string,
			refresh?: boolean,
			botFilter?: BotFilterOptions,
		) =>
			request<CycleTimeMetrics>(
				`/metrics/cycle-time?${buildMetricsParams(repositories, start, end, refresh, botFilter)}`,
			),
		reviews: (
			repositories?: string[],
			start?: string,
			end?: string,
			refresh?: boolean,
			botFilter?: BotFilterOptions,
		) =>
			request<ReviewMetrics>(
				`/metrics/reviews?${buildMetricsParams(repositories, start, end, refresh, botFilter)}`,
			),
		dora: (
			repositories?: string[],
			start?: string,
			end?: string,
			refresh?: boolean,
			botFilter?: BotFilterOptions,
		) =>
			request<DORAMetrics>(
				`/metrics/dora?${buildMetricsParams(repositories, start, end, refresh, botFilter)}`,
			),
		productivityScore: (
			repositories?: string[],
			start?: string,
			end?: string,
			refresh?: boolean,
			botFilter?: BotFilterOptions,
		) =>
			request<ProductivityScore>(
				`/metrics/productivity-score?${buildMetricsParams(repositories, start, end, refresh, botFilter)}`,
			),
		daily: (repositories?: string[], start?: string, end?: string, refresh?: boolean) =>
			request<DailyMetrics[]>(
				`/metrics/daily?${buildMetricsParams(repositories, start, end, refresh)}`,
			),
		pullRequests: (repositories?: string[], start?: string, end?: string) =>
			request<MemberPullRequest[]>(
				`/metrics/pull-requests?${buildDateRangeParams(repositories, start, end)}`,
			),
	},

	// Sprints
	sprints: {
		list: (repository: string) => request<Sprint[]>(`/sprints?repository=${repository}`),
		create: (sprint: Omit<Sprint, 'id'>) =>
			request<Sprint>('/sprints', { method: 'POST', body: sprint }),
		get: (id: string) => request<Sprint>(`/sprints/${id}`),
		getPerformance: (id: string) => request<SprintPerformance>(`/sprints/${id}/performance`),
	},

	// Bot Users
	botUsers: {
		list: () => request<BotUser[]>('/bot-users'),
		add: (username: string) =>
			request<BotUser>('/bot-users', { method: 'POST', body: { username } }),
		delete: (username: string) =>
			request<void>(`/bot-users?username=${encodeURIComponent(username)}`, { method: 'DELETE' }),
	},

	// Team
	team: {
		listMembers: (botFilter?: BotFilterOptions) => {
			const params = new URLSearchParams();
			if (botFilter?.excludeBots === false) params.append('exclude_bots', 'false');
			if (botFilter?.botsOnly) params.append('bots_only', 'true');
			const qs = params.toString();
			return request<TeamMember[]>(`/team/members${qs ? `?${qs}` : ''}`);
		},
		getMemberStats: (id: string, repositories?: string[], start?: string, end?: string) =>
			request<MemberStats>(
				`/team/members/${id}/stats?${buildDateRangeParams(repositories, start, end)}`,
			),
		getMemberPullRequests: (id: string, repositories?: string[], start?: string, end?: string) =>
			request<MemberPullRequest[]>(
				`/team/members/${id}/pull-requests?${buildDateRangeParams(repositories, start, end)}`,
			),
		getMemberReviews: (id: string, repositories?: string[], start?: string, end?: string) =>
			request<MemberReview[]>(
				`/team/members/${id}/reviews?${buildDateRangeParams(repositories, start, end)}`,
			),
	},
};
