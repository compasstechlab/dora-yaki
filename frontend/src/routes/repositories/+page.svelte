<script lang="ts">
import { onMount } from 'svelte';
import {
	api,
	type BatchAddResult,
	type DataDateRange,
	type GitHubMe,
	type GitHubOrgRepo,
	type SyncResponse,
} from '$api/client';
import { locale, t } from '$i18n';
import { repositories, selectedRepositories } from '$stores/repositories';

// Data date ranges
let dateRanges = $state<Record<string, DataDateRange>>({});

async function fetchDateRanges() {
	try {
		const ranges = await api.repositories.dateRanges();
		const map: Record<string, DataDateRange> = {};
		for (const r of ranges ?? []) {
			map[r.repositoryId] = r;
		}
		dateRanges = map;
	} catch (e) {
		console.error('Failed to fetch date ranges:', e);
	}
}

// Authenticated user info
let githubMe = $state<GitHubMe | null>(null);
let ownerOptions = $state<string[]>([]);

// Owner search
let selectedOwner = $state('');
let customOwner = $state('');
let repoType = $state('');
// Build ownerName from selection
let ownerName = $derived(selectedOwner === '__custom__' ? customOwner : selectedOwner);
let ownerRepos = $state<GitHubOrgRepo[]>([]);
let hasSearched = $state(false);
let isLoadingOwner = $state(false);
let searchError = $state('');
let selectedRepos = $state<Set<string>>(new Set());
let isBatchAdding = $state(false);
let batchResults = $state<BatchAddResult[]>([]);

onMount(async () => {
	fetchDateRanges();
	try {
		githubMe = await api.github.getMe();
		// Set owner options from username + orgs
		ownerOptions = [githubMe.login, ...(githubMe.orgs || [])];
		// Select first owner by default
		if (ownerOptions.length > 0) {
			selectedOwner = ownerOptions[0];
		}
	} catch (e) {
		console.error('Failed to get GitHub user info:', e);
	}
});

// Sync state
let isSyncing = $state<Record<string, boolean>>({});
let syncResults = $state<Record<string, SyncResponse | null>>({});
let isSyncingAll = $state(false);
let syncAllProgress = $state({ done: 0, total: 0 });

// Full names of registered repositories
let registeredNames = $derived(new Set($repositories.map((r) => r.fullName)));

// Filtered list (excluding registered and archived)
let filteredRepos = $derived(
	ownerRepos.filter((r) => !r.archived && !registeredNames.has(r.fullName)),
);
let archivedCount = $derived(ownerRepos.filter((r) => r.archived).length);
let registeredCount = $derived(
	ownerRepos.filter((r) => !r.archived && registeredNames.has(r.fullName)).length,
);

async function fetchOwnerRepos() {
	if (!ownerName) {
		searchError = $t('repositories.enterOwner');
		return;
	}

	isLoadingOwner = true;
	searchError = '';
	ownerRepos = [];
	hasSearched = false;
	selectedRepos = new Set();
	batchResults = [];

	try {
		ownerRepos = await api.github.listOwnerRepos(ownerName, repoType || undefined) ?? [];
		hasSearched = true;
	} catch (e) {
		searchError = e instanceof Error ? e.message : $t('repositories.fetchFailed');
	} finally {
		isLoadingOwner = false;
	}
}

function toggleRepo(fullName: string) {
	const next = new Set(selectedRepos);
	if (next.has(fullName)) {
		next.delete(fullName);
	} else {
		next.add(fullName);
	}
	selectedRepos = next;
}

function selectAll() {
	selectedRepos = new Set(filteredRepos.map((r) => r.fullName));
}

function deselectAll() {
	selectedRepos = new Set();
}

async function batchAddRepositories() {
	if (selectedRepos.size === 0) return;

	isBatchAdding = true;
	batchResults = [];

	const reposToAdd = filteredRepos
		.filter((r) => selectedRepos.has(r.fullName))
		.map((r) => ({ owner: r.owner, name: r.name }));

	try {
		batchResults = await repositories.batchAdd(reposToAdd);
		// Remove successful ones from selection
		const successNames = batchResults.filter((r) => r.success).map((r) => `${r.owner}/${r.name}`);
		const next = new Set(selectedRepos);
		for (const n of successNames) {
			next.delete(n);
		}
		selectedRepos = next;
	} catch (e) {
		searchError = e instanceof Error ? e.message : $t('repositories.batchFailed');
	} finally {
		isBatchAdding = false;
	}
}

// Dropdown menu toggle
let openMenuId = $state<string | null>(null);

function handleWindowClick() {
	openMenuId = null;
}

function toggleMenu(id: string, event: MouseEvent) {
	event.stopPropagation();
	openMenuId = openMenuId === id ? null : id;
}

async function syncRepository(id: string, range: string = 'week') {
	openMenuId = null;
	isSyncing[id] = true;
	syncResults[id] = null;

	try {
		const result = await repositories.sync(id, range);
		syncResults[id] = result;
		fetchDateRanges();
	} catch (e) {
		console.error('Sync failed:', e);
	} finally {
		isSyncing[id] = false;
	}
}

async function syncAllRepositories(range: string = 'week') {
	openMenuId = null;
	const repos = $repositories;
	if (repos.length === 0) return;

	isSyncingAll = true;
	syncAllProgress = { done: 0, total: repos.length };

	for (const repo of repos) {
		isSyncing[repo.id] = true;
		syncResults[repo.id] = null;
		try {
			const result = await repositories.sync(repo.id, range);
			syncResults[repo.id] = result;
		} catch (e) {
			console.error(`Sync failed for ${repo.fullName}:`, e);
		} finally {
			isSyncing[repo.id] = false;
			syncAllProgress.done++;
		}
	}

	isSyncingAll = false;
}

async function removeRepository(id: string) {
	if (!confirm($t('common.confirmDelete'))) return;

	try {
		await repositories.remove(id);
		// Remove from selection if selected
		if ($selectedRepositories.includes(id)) {
			$selectedRepositories = $selectedRepositories.filter((r) => r !== id);
		}
	} catch (e) {
		console.error('Remove failed:', e);
	}
}

function toggleRepoSelection(id: string) {
	if ($selectedRepositories.includes(id)) {
		const next = $selectedRepositories.filter((r) => r !== id);
		$selectedRepositories = next.length === 0 ? [] : next;
	} else {
		$selectedRepositories = [...$selectedRepositories, id];
	}
}

let isRepoSelected = $derived(
	(id: string) => $selectedRepositories.length === 0 || $selectedRepositories.includes(id),
);
</script>

<svelte:head>
	<title>{$t('repositories.pageTitle')}</title>
</svelte:head>

<svelte:window onclick={handleWindowClick} />

<div class="page">
	<header class="page-header">
		<h1>{$t('repositories.title')}</h1>
	</header>

	<!-- Repository Search & Add -->
	<section class="section">
		<div class="card">
			<h2 class="card-title">{$t('repositories.addRepo')}</h2>
			<p class="card-desc">{$t('repositories.addRepoDesc')}</p>
			<form class="add-form" onsubmit={(e) => { e.preventDefault(); fetchOwnerRepos(); }}>
				<div class="form-row">
					{#if ownerOptions.length > 0}
						<select bind:value={selectedOwner} class="input select-owner" disabled={isLoadingOwner}>
							{#each ownerOptions as opt}
								<option value={opt}>{opt}</option>
							{/each}
							<option value="__custom__">{$t('repositories.otherManual')}</option>
						</select>
					{/if}
					{#if ownerOptions.length === 0 || selectedOwner === '__custom__'}
						<input
							type="text"
							bind:value={customOwner}
							placeholder={$t('repositories.ownerPlaceholder')}
							class="input"
							disabled={isLoadingOwner}
						>
					{/if}
					<select bind:value={repoType} class="input select" disabled={isLoadingOwner}>
						<option value="">{$t('repositories.allTypes')}</option>
						<option value="public">Public</option>
						<option value="private">Private</option>
					</select>
					<button type="submit" class="btn btn-primary" disabled={isLoadingOwner}>
						{isLoadingOwner ? $t('common.searching') : $t('common.search')}
					</button>
				</div>
				{#if searchError}
					<p class="error">{searchError}</p>
				{/if}
			</form>

			{#if hasSearched && ownerRepos.length === 0 && !searchError}
				<p class="empty-message">{$t('repositories.noSearchResults', { owner: ownerName })}</p>
			{/if}

			{#if ownerRepos.length > 0}
				<div class="search-results">
					<div class="results-actions">
						<span class="results-count">
							{$t('repositories.resultsShowing', { count: filteredRepos.length })}
							{#if archivedCount > 0}
								{$t('repositories.archivedExcluded', { count: archivedCount })}
							{/if}
							{#if registeredCount > 0}
								{$t('repositories.registeredExcluded', { count: registeredCount })}
							{/if}
						</span>
						<div class="results-buttons">
							<button class="btn btn-secondary btn-sm" onclick={selectAll}>
								{$t('common.selectAll')}
							</button>
							<button class="btn btn-secondary btn-sm" onclick={deselectAll}>
								{$t('common.deselectAll')}
							</button>
						</div>
					</div>

					<div class="search-repo-list">
						{#each filteredRepos as repo}
							<label class="search-repo-item" class:checked={selectedRepos.has(repo.fullName)}>
								<input
									type="checkbox"
									checked={selectedRepos.has(repo.fullName)}
									onchange={() => toggleRepo(repo.fullName)}
								>
								<div class="search-repo-info">
									<span class="search-repo-name">{repo.name}</span>
									{#if repo.private}
										<span class="badge badge-private">Private</span>
									{:else}
										<span class="badge badge-public">Public</span>
									{/if}
									{#if repo.language}
										<span class="search-repo-lang">{repo.language}</span>
									{/if}
								</div>
								{#if repo.description}
									<p class="search-repo-desc">{repo.description}</p>
								{/if}
							</label>
						{/each}
					</div>

					{#if filteredRepos.length > 0}
						<div class="batch-actions">
							<button
								class="btn btn-primary"
								disabled={selectedRepos.size === 0 || isBatchAdding}
								onclick={batchAddRepositories}
							>
								{isBatchAdding
									? $t('common.adding')
									: $t('repositories.addSelected', { count: selectedRepos.size })}
							</button>
						</div>
					{/if}

					{#if batchResults.length > 0}
						<div class="batch-results">
							<h3 class="batch-results-title">{$t('repositories.addResults')}</h3>
							{#each batchResults as result}
								<div
									class="batch-result-item"
									class:success={result.success}
									class:failure={!result.success}
								>
									<span>{result.owner}/{result.name}</span>
									{#if result.success}
										<span class="result-badge result-success">{$t('common.success')}</span>
									{:else}
										<span class="result-badge result-failure">{result.error}</span>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</section>

	<!-- Repository List -->
	<section class="section">
		<div class="section-header">
			<h2 class="section-title">{$t('repositories.registeredRepos')}</h2>
			{#if $repositories.length > 0}
				<div class="section-header-actions">
					{#if isSyncingAll}
						<span class="sync-all-progress">
							{$t('repositories.syncProgress', { done: syncAllProgress.done, total: syncAllProgress.total })}
						</span>
					{/if}
					<div class="split-btn-group">
						<button
							class="btn btn-primary split-btn-main"
							onclick={() => syncAllRepositories('week')}
							disabled={isSyncingAll}
						>
							{isSyncingAll ? $t('repositories.syncAllInProgress') : $t('repositories.syncAll', { range: 'week' })}
						</button>
						<button
							class="btn btn-primary split-btn-toggle"
							onclick={(e) => toggleMenu('__sync_all__', e)}
							disabled={isSyncingAll}
						>
							▼
						</button>
						{#if openMenuId === '__sync_all__'}
							<!-- svelte-ignore a11y_click_events_have_key_events -->
							<!-- svelte-ignore a11y_no_static_element_interactions -->
							<div class="split-btn-menu" onclick={(e) => { e.stopPropagation(); }}>
								<button class="split-btn-menu-item" onclick={() => syncAllRepositories('day')}>
									{$t('repositories.syncAll', { range: 'day' })}
									<span class="menu-hint">{$t('repositories.dayHint')}</span>
								</button>
								<button class="split-btn-menu-item" onclick={() => syncAllRepositories('month')}>
									{$t('repositories.syncAll', { range: 'month' })}
									<span class="menu-hint">{$t('repositories.monthHint')}</span>
								</button>
								<button class="split-btn-menu-item" onclick={() => syncAllRepositories('6month')}>
									{$t('repositories.syncAll', { range: '6month' })}
									<span class="menu-hint">{$t('repositories.6monthHint')}</span>
								</button>
								<button class="split-btn-menu-item" onclick={() => syncAllRepositories('year')}>
									{$t('repositories.syncAll', { range: 'year' })}
									<span class="menu-hint">{$t('repositories.yearHint')}</span>
								</button>
								<button class="split-btn-menu-item" onclick={() => syncAllRepositories('full')}>
									{$t('repositories.syncAll', { range: 'full' })}
									<span class="menu-hint">{$t('repositories.fullHint')}</span>
								</button>
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>

		{#if $repositories.length === 0}
			<div class="empty-state">
				<p>{$t('repositories.noRepos')}</p>
			</div>
		{:else}
			<div class="repo-list">
				{#each $repositories as repo}
					<div class="repo-card" class:selected={isRepoSelected(repo.id)}>
						<div class="repo-info">
							<h3 class="repo-name">{repo.fullName}</h3>
							<p class="repo-meta">
								{#if repo.private}
									<span class="badge badge-private">Private</span>
								{:else}
									<span class="badge badge-public">Public</span>
								{/if}
								<span class="updated">
									{$t('repositories.updated', { date: new Date(repo.updatedAt).toLocaleDateString($locale) })}
								</span>
								{#if repo.lastSyncedAt}
									<span class="synced">
										{$t('repositories.lastSynced', { date: new Date(repo.lastSyncedAt).toLocaleString($locale, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }) })}
									</span>
								{:else}
									<span class="synced not-synced">{$t('repositories.notSynced')}</span>
								{/if}
							</p>
							{#if dateRanges[repo.id]}
								<p class="repo-data-range">
									{#if dateRanges[repo.id].oldestDate && dateRanges[repo.id].newestDate}
										<span class="data-range">
											{$t('repositories.dataRange', {
												oldest: new Date(dateRanges[repo.id].oldestDate ?? '').toLocaleDateString($locale),
												newest: new Date(dateRanges[repo.id].newestDate ?? '').toLocaleDateString($locale)
											})}
										</span>
										<span class="pr-count">
											{$t('repositories.prCount', { count: dateRanges[repo.id].prCount })}
										</span>
									{:else}
										<span class="data-range no-data">{$t('repositories.dataRangeNoData')}</span>
									{/if}
								</p>
							{/if}
						</div>

						<div class="repo-actions">
							<button class="btn btn-secondary" onclick={() => toggleRepoSelection(repo.id)}>
								{$selectedRepositories.includes(repo.id) ? $t('common.deselect') : $t('common.select')}
							</button>
							<div class="split-btn-group">
								<button
									class="btn btn-primary split-btn-main"
									onclick={() => syncRepository(repo.id, 'week')}
									disabled={isSyncing[repo.id]}
								>
									{isSyncing[repo.id] ? $t('common.syncing') : $t('repositories.syncRange', { range: 'week' })}
								</button>
								<button
									class="btn btn-primary split-btn-toggle"
									onclick={(e) => toggleMenu(repo.id, e)}
									disabled={isSyncing[repo.id]}
								>
									▼
								</button>
								{#if openMenuId === repo.id}
									<!-- svelte-ignore a11y_click_events_have_key_events -->
									<!-- svelte-ignore a11y_no_static_element_interactions -->
									<div class="split-btn-menu" onclick={(e) => { e.stopPropagation(); }}>
										<button
											class="split-btn-menu-item"
											onclick={() => syncRepository(repo.id, 'day')}
										>
											{$t('repositories.daySync')}
											<span class="menu-hint">{$t('repositories.dayHint')}</span>
										</button>
										<button
											class="split-btn-menu-item"
											onclick={() => syncRepository(repo.id, 'month')}
										>
											{$t('repositories.monthSync')}
											<span class="menu-hint">{$t('repositories.monthHint')}</span>
										</button>
										<button
											class="split-btn-menu-item"
											onclick={() => syncRepository(repo.id, '6month')}
										>
											{$t('repositories.6monthSync')}
											<span class="menu-hint">{$t('repositories.6monthHint')}</span>
										</button>
										<button
											class="split-btn-menu-item"
											onclick={() => syncRepository(repo.id, 'year')}
										>
											{$t('repositories.yearSync')}
											<span class="menu-hint">{$t('repositories.yearHint')}</span>
										</button>
										<button
											class="split-btn-menu-item"
											onclick={() => syncRepository(repo.id, 'full')}
										>
											{$t('repositories.fullSync')}
											<span class="menu-hint">{$t('repositories.fullHint')}</span>
										</button>
									</div>
								{/if}
							</div>
							<button class="btn btn-danger" onclick={() => removeRepository(repo.id)}>
								{$t('common.delete')}
							</button>
						</div>

						{#if syncResults[repo.id]}
							<div class="sync-result">
								<p>{$t('common.syncComplete')}</p>
								<ul>
									<li>
										{$t('repositories.syncResult.pr', { count: syncResults[repo.id]?.pullRequests ?? 0 })}
									</li>
									<li>
										{$t('repositories.syncResult.reviews', { count: syncResults[repo.id]?.reviews ?? 0 })}
									</li>
									<li>
										{$t('repositories.syncResult.deployments', { count: syncResults[repo.id]?.deployments ?? 0 })}
									</li>
									<li>
										{$t('repositories.syncResult.members', { count: syncResults[repo.id]?.teamMembers ?? 0 })}
									</li>
								</ul>
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</section>
</div>

<style>
.page {
	max-width: 1000px;
	margin: 0 auto;
}

.page-header {
	margin-bottom: 2rem;
}

.section {
	margin-bottom: 2rem;
}

.section-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-bottom: 1rem;
	gap: 1rem;
}

.section-header-actions {
	display: flex;
	align-items: center;
	gap: 0.75rem;
}

.section-title {
	font-size: 1.125rem;
	margin-bottom: 0;
}

.sync-all-progress {
	font-size: 0.8125rem;
	color: var(--color-text-muted);
	white-space: nowrap;
}

.card-title {
	font-size: 1rem;
	margin-bottom: 0.25rem;
}

.card-desc {
	font-size: 0.8125rem;
	color: var(--color-text-muted);
	margin-bottom: 1rem;
}

/* Form */
.add-form {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.form-row {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.input {
	flex: 1;
	padding: 0.625rem 0.875rem;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	font-size: 0.875rem;
}

.input:focus {
	outline: none;
	border-color: var(--color-primary);
	box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.select-owner {
	flex: 1;
	min-width: 160px;
}

.select {
	flex: 0 0 auto;
	min-width: 120px;
}

.error {
	color: var(--color-danger);
	font-size: 0.875rem;
}

.empty-message {
	color: var(--color-text-muted, #6b7280);
	font-size: 0.875rem;
	margin-top: 1rem;
}

/* Search Results */
.search-results {
	margin-top: 1.5rem;
}

.results-actions {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-bottom: 1rem;
}

.results-count {
	font-size: 0.875rem;
	color: var(--color-text-muted);
}

.results-buttons {
	display: flex;
	gap: 0.5rem;
}

.btn-sm {
	padding: 0.375rem 0.75rem;
	font-size: 0.75rem;
}

.search-repo-list {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
	max-height: 400px;
	overflow-y: auto;
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	padding: 0.5rem;
}

.search-repo-item {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: 0.5rem;
	padding: 0.625rem 0.75rem;
	border-radius: var(--radius-md);
	cursor: pointer;
	transition: background 0.15s;
}

.search-repo-item:hover {
	background: var(--color-bg-hover, rgba(0, 0, 0, 0.03));
}

.search-repo-item.checked {
	background: rgba(59, 130, 246, 0.05);
}

.search-repo-info {
	display: flex;
	align-items: center;
	gap: 0.5rem;
}

.search-repo-name {
	font-weight: 500;
	font-size: 0.875rem;
}

.search-repo-lang {
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.search-repo-desc {
	width: 100%;
	margin: 0;
	padding-left: 1.75rem;
	font-size: 0.75rem;
	color: var(--color-text-muted);
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

/* Batch Add */
.batch-actions {
	margin-top: 1rem;
	display: flex;
	justify-content: flex-end;
}

.batch-results {
	margin-top: 1rem;
	padding: 1rem;
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
}

.batch-results-title {
	font-size: 0.875rem;
	margin-bottom: 0.75rem;
}

.batch-result-item {
	display: flex;
	align-items: center;
	justify-content: space-between;
	padding: 0.375rem 0;
	font-size: 0.875rem;
	border-bottom: 1px solid var(--color-border);
}

.batch-result-item:last-child {
	border-bottom: none;
}

.result-badge {
	font-size: 0.75rem;
	padding: 0.125rem 0.5rem;
	border-radius: var(--radius-sm);
}

.result-success {
	background: rgba(16, 185, 129, 0.1);
	color: var(--color-success);
}

.result-failure {
	background: rgba(239, 68, 68, 0.1);
	color: var(--color-danger);
}

/* Repository List */
.empty-state {
	text-align: center;
	padding: 3rem;
	background: var(--color-bg-card);
	border-radius: var(--radius-lg);
	border: 1px solid var(--color-border);
	color: var(--color-text-muted);
}

.repo-list {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.repo-card {
	background: var(--color-bg-card);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-lg);
	padding: 1.25rem;
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: 1rem;
}

.repo-card.selected {
	border-color: var(--color-primary);
	box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.repo-info {
	flex: 1;
	min-width: 200px;
}

.repo-name {
	font-size: 1rem;
	margin-bottom: 0.25rem;
}

.repo-meta {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.badge {
	padding: 0.125rem 0.5rem;
	border-radius: var(--radius-sm);
	font-weight: 500;
}

.badge-private {
	background: rgba(239, 68, 68, 0.1);
	color: var(--color-danger);
}

.badge-public {
	background: rgba(16, 185, 129, 0.1);
	color: var(--color-success);
}

.synced {
	font-size: 0.75rem;
	color: var(--color-text-muted);
}

.synced.not-synced {
	color: var(--color-warning, #f59e0b);
}

.repo-data-range {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	font-size: 0.75rem;
	color: var(--color-text-muted);
	margin: 0.25rem 0 0;
}

.data-range.no-data {
	color: var(--color-text-muted);
	opacity: 0.6;
}

.pr-count {
	font-size: 0.6875rem;
	padding: 0.0625rem 0.375rem;
	background: rgba(59, 130, 246, 0.08);
	border-radius: var(--radius-sm);
	color: var(--color-primary, #3b82f6);
}

.repo-actions {
	display: flex;
	gap: 0.5rem;
}

/* Split Button */
.split-btn-group {
	position: relative;
	display: inline-flex;
}

.split-btn-main {
	border-top-right-radius: 0;
	border-bottom-right-radius: 0;
}

.split-btn-toggle {
	border-top-left-radius: 0;
	border-bottom-left-radius: 0;
	border-left: 1px solid rgba(255, 255, 255, 0.2);
	padding: 0.5rem 0.5rem;
	font-size: 0.625rem;
}

.split-btn-menu {
	position: absolute;
	top: 100%;
	right: 0;
	margin-top: 0.25rem;
	background: var(--color-bg-card, #fff);
	border: 1px solid var(--color-border);
	border-radius: var(--radius-md);
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
	z-index: 10;
	min-width: 160px;
	overflow: hidden;
}

.split-btn-menu-item {
	display: block;
	width: 100%;
	padding: 0.5rem 0.75rem;
	border: none;
	background: none;
	font-size: 0.8125rem;
	text-align: left;
	cursor: pointer;
	color: var(--color-text, inherit);
}

.split-btn-menu-item:hover {
	background: var(--color-bg-hover, rgba(0, 0, 0, 0.04));
}

.menu-hint {
	color: var(--color-text-muted);
	font-size: 0.75rem;
}

.btn-danger {
	background: var(--color-danger);
	color: white;
}

.btn-danger:hover {
	background: #dc2626;
}

.sync-result {
	width: 100%;
	margin-top: 0.5rem;
	padding: 0.75rem;
	background: rgba(16, 185, 129, 0.05);
	border-radius: var(--radius-md);
	font-size: 0.875rem;
}

.sync-result ul {
	display: flex;
	gap: 1rem;
	margin: 0.5rem 0 0;
	padding: 0;
	list-style: none;
}

@media (max-width: 640px) {
	.form-row {
		flex-direction: column;
	}

	.input {
		width: 100%;
	}

	.select {
		min-width: unset;
	}

	.repo-card {
		flex-direction: column;
		align-items: stretch;
	}

	.repo-actions {
		justify-content: flex-end;
	}

	.results-actions {
		flex-direction: column;
		gap: 0.5rem;
		align-items: flex-start;
	}
}
</style>
