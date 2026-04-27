<script lang="ts">
import type { Snippet } from 'svelte';
import { onMount } from 'svelte';
import FlashMessage from '$components/FlashMessage.svelte';
import PeriodSelector from '$components/PeriodSelector.svelte';
import { LOCALE_NAMES, LOCALES, type Locale, locale, t } from '$i18n';
import { authChecked, authUser, checkAuth, logout } from '$stores/auth';
import { repositories, selectedRepositories } from '$stores/repositories';
import '../app.css';

let { children }: { children: Snippet } = $props();

let currentPath = $state('/');
let isLoginRoute = $state(false);

onMount(async () => {
	currentPath = window.location.pathname;
	isLoginRoute = currentPath === '/login';

	if (isLoginRoute) {
		// Login page handles its own auth check; do not redirect.
		return;
	}

	const user = await checkAuth();
	if (!user) {
		window.location.href = '/login';
		return;
	}
	locale.init();
	await repositories.load();
});

async function handleLogout() {
	await logout();
	window.location.href = '/login';
}

let navItems = $derived([
	{ path: '/', label: $t('nav.dashboard'), icon: '📊' },
	{ path: '/metrics', label: $t('nav.metrics'), icon: '📈' },
	{ path: '/repo', label: $t('nav.repo'), icon: '📁' },
	{ path: '/team', label: $t('nav.team'), icon: '👥' },
	{ path: '/bots', label: $t('nav.bots'), icon: '🤖' },
	{ path: '/repositories', label: $t('nav.repositories'), icon: '⚙️' },
]);

// "All" checkbox state (empty array = all repos selected)
let isAllSelected = $derived($selectedRepositories.length === 0);

function toggleAll() {
	if (isAllSelected) {
		if ($repositories.length > 0) {
			$selectedRepositories = [$repositories[0].id];
		}
	} else {
		$selectedRepositories = [];
	}
}

function toggleRepo(repoId: string) {
	if ($selectedRepositories.includes(repoId)) {
		const next = $selectedRepositories.filter((id) => id !== repoId);
		$selectedRepositories = next.length === 0 ? [] : next;
	} else {
		$selectedRepositories = [...$selectedRepositories, repoId];
	}
}

let selectionLabel = $derived(
	$selectedRepositories.length === 0
		? $t('common.all')
		: $t('common.selected', { count: $selectedRepositories.length }),
);

function handleLocaleChange(e: Event) {
	const target = e.target as HTMLSelectElement;
	locale.set(target.value as Locale);
}
</script>

<FlashMessage />

{#if isLoginRoute}
	{@render children()}
{:else if !$authChecked}
	<div class="loading">Loading…</div>
{:else if !$authUser}
	<div class="loading">Redirecting…</div>
{:else}
	<div class="layout">
		<aside class="sidebar">
			<div class="logo">
				<span class="logo-icon">📊</span>
				<span class="logo-text">DORA-yaki</span>
			</div>

			<nav class="nav">
				{#each navItems as item}<a
					href={item.path}
					class="nav-item"
					class:active={item.path === "/"
            ? currentPath === "/"
            : currentPath === item.path ||
              currentPath.startsWith(item.path + "/")}
					onclick={() => (currentPath = item.path)}
				>
					<span class="nav-icon">{item.icon}</span>
					<span class="nav-label">{item.label}</span>
				</a>{/each}
			</nav>

			<div class="sidebar-footer">
				{#if $authUser}
					<div class="user-info">
						{#if $authUser.avatarUrl}
							<img
								src={$authUser.avatarUrl}
								alt=""
								class="user-avatar"
								referrerpolicy="no-referrer"
							>
						{/if}
						<span class="user-name" title={$authUser.name || $authUser.login}>
							{$authUser.login}
						</span>
						<button class="logout-btn" onclick={handleLogout} title="Logout">⏻</button>
					</div>
				{/if}

				<div class="lang-selector">
					<span class="lang-label"><span class="lang-icon">🌐</span> {$t("language.label")}</span>
					<select class="lang-select" value={$locale} onchange={handleLocaleChange}>
						{#each LOCALES as loc}
							<option value={loc}>{LOCALE_NAMES[loc]}</option>
						{/each}
					</select>
				</div>

				<PeriodSelector />

				<div class="repo-selector">
					<div class="repo-selector-header">
						<span class="repo-selector-title">{$t("common.repository")}</span>
						<span class="repo-selector-badge">{selectionLabel}</span>
					</div>
					<label class="repo-checkbox repo-checkbox-all">
						<input type="checkbox" checked={isAllSelected} onchange={toggleAll}>
						<span>{$t("common.all")}</span>
					</label>
					{#each $repositories as repo}
						<label class="repo-checkbox">
							<input
								type="checkbox"
								checked={isAllSelected || $selectedRepositories.includes(repo.id)}
								disabled={isAllSelected}
								onchange={() => toggleRepo(repo.id)}
							>
							<span>{repo.fullName}</span>
						</label>
					{/each}
				</div>
			</div>
		</aside>

		<main class="main">
			{@render children()}
			<footer class="app-footer">
				<a
					href="https://github.com/compasstechlab/dora-yaki"
					target="_blank"
					rel="noopener noreferrer"
				>
					{$t('footer.sourceCode')}
				</a>
				<span class="footer-separator">|</span>
				<span>{$t('footer.license')}</span>
			</footer>
		</main>
	</div>
{/if}

<style>
.layout {
	display: flex;
	min-height: 100vh;
}

.sidebar {
	width: 240px;
	background: var(--color-bg-sidebar);
	color: white;
	display: flex;
	flex-direction: column;
	position: fixed;
	top: 0;
	left: 0;
	bottom: 0;
	z-index: 100;
}

.logo {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 1.5rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.logo-icon {
	font-size: 1.5rem;
}

.logo-text {
	font-size: 1.25rem;
	font-weight: 700;
}

.nav {
	flex: 1;
	padding: 1rem 0;
}

.nav-item {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 0.75rem 1.5rem;
	color: rgba(255, 255, 255, 0.7);
	text-decoration: none;
	transition: all 0.15s ease;
}

.nav-item:hover {
	background: rgba(255, 255, 255, 0.1);
	color: white;
	text-decoration: none;
}

.nav-item.active {
	background: rgba(59, 130, 246, 0.2);
	color: white;
	border-right: 3px solid var(--color-primary);
}

.nav-icon {
	font-size: 1.125rem;
}

.nav-label {
	font-size: 0.875rem;
	font-weight: 500;
}

.sidebar-footer {
	padding: 0.75rem 1rem;
	border-top: 1px solid rgba(255, 255, 255, 0.1);
	max-height: 40vh;
	overflow-y: auto;
}

.user-info {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.5rem;
	margin-bottom: 0.75rem;
	background: rgba(255, 255, 255, 0.05);
	border-radius: var(--radius-sm);
}

.user-avatar {
	width: 28px;
	height: 28px;
	border-radius: 50%;
	flex-shrink: 0;
}

.user-name {
	flex: 1;
	font-size: 0.8125rem;
	color: rgba(255, 255, 255, 0.85);
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.logout-btn {
	background: rgba(255, 255, 255, 0.1);
	color: rgba(255, 255, 255, 0.7);
	border: none;
	border-radius: var(--radius-sm);
	width: 26px;
	height: 26px;
	font-size: 0.875rem;
	cursor: pointer;
	flex-shrink: 0;
	transition:
		background 0.15s ease,
		color 0.15s ease;
}

.logout-btn:hover {
	background: rgba(239, 68, 68, 0.4);
	color: white;
}

.loading {
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	color: var(--color-text-muted, #6b7280);
	font-size: 0.875rem;
}

.lang-selector {
	display: flex;
	align-items: center;
	justify-content: space-between;
	margin-bottom: 0.75rem;
	padding-bottom: 0.75rem;
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.lang-label {
	font-size: 0.75rem;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	color: rgba(255, 255, 255, 0.5);
	display: flex;
	align-items: center;
	gap: 0.25rem;
}

.lang-icon {
	font-size: 1.125rem;
}

.lang-select {
	background: rgba(255, 255, 255, 0.1);
	color: white;
	border: 1px solid rgba(255, 255, 255, 0.2);
	border-radius: var(--radius-sm);
	padding: 0.25rem 0.5rem;
	font-size: 0.75rem;
	cursor: pointer;
}

.lang-select option {
	background: var(--color-bg-sidebar);
	color: white;
}

.repo-selector {
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
}

.repo-selector-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 0.5rem;
}

.repo-selector-title {
	font-size: 0.75rem;
	text-transform: uppercase;
	letter-spacing: 0.05em;
	color: rgba(255, 255, 255, 0.5);
}

.repo-selector-badge {
	font-size: 0.625rem;
	padding: 0.125rem 0.375rem;
	background: rgba(59, 130, 246, 0.3);
	border-radius: var(--radius-sm);
	color: rgba(255, 255, 255, 0.8);
}

.repo-checkbox {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	padding: 0.375rem 0.5rem;
	border-radius: var(--radius-sm);
	cursor: pointer;
	font-size: 0.8125rem;
	color: rgba(255, 255, 255, 0.7);
	transition: background 0.15s ease;
}

.repo-checkbox:hover {
	background: rgba(255, 255, 255, 0.1);
}

.repo-checkbox-all {
	border-bottom: 1px solid rgba(255, 255, 255, 0.1);
	padding-bottom: 0.5rem;
	margin-bottom: 0.25rem;
	font-weight: 500;
	color: rgba(255, 255, 255, 0.9);
}

.repo-checkbox input[type="checkbox"] {
	accent-color: var(--color-primary);
	width: 14px;
	height: 14px;
}

.repo-checkbox span {
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.main {
	flex: 1;
	margin-left: 240px;
	padding: 2rem;
	min-height: 100vh;
}

.app-footer {
	margin-top: 3rem;
	padding: 1.5rem 0;
	border-top: 1px solid var(--color-border, #e5e7eb);
	text-align: center;
	font-size: 0.75rem;
	color: var(--color-text-muted, #6b7280);
}

.app-footer a {
	color: var(--color-primary, #3b82f6);
	text-decoration: none;
}

.app-footer a:hover {
	text-decoration: underline;
}

.footer-separator {
	margin: 0 0.5rem;
	color: var(--color-text-muted, #9ca3af);
}

@media (max-width: 768px) {
	.sidebar {
		width: 100%;
		height: auto;
		position: relative;
	}

	.main {
		margin-left: 0;
	}
}
</style>
