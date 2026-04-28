<script lang="ts">
import { onMount } from 'svelte';
import { checkAuth, redirectToGitHubLogin } from '$stores/auth';

let error = $state<string | null>(null);
let loading = $state(true);

onMount(async () => {
	const params = new URLSearchParams(window.location.search);
	error = params.get('error');
	const user = await checkAuth();
	if (user) {
		window.location.href = '/';
		return;
	}
	loading = false;
});

function handleLogin() {
	redirectToGitHubLogin('/');
}
</script>

<main class="login">
	<div class="card">
		<div class="logo">
			<span class="logo-icon">📊</span>
			<h1>DORA-yaki</h1>
		</div>
		<p class="subtitle">Sign in with your GitHub account to continue.</p>

		{#if error}
			<div class="error" role="alert">Login failed: {error}</div>
		{/if}

		{#if !loading}
			<button class="github-btn" onclick={handleLogin}>
				<svg viewBox="0 0 16 16" width="20" height="20" fill="currentColor" aria-hidden="true">
					<path
						d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"
					/>
				</svg>
				Sign in with GitHub
			</button>
		{/if}
	</div>
</main>

<style>
.login {
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	background: var(--color-bg-sidebar, #1f2937);
	padding: 1rem;
}

.card {
	background: white;
	padding: 2.5rem;
	border-radius: 0.75rem;
	box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.2);
	max-width: 400px;
	width: 100%;
	text-align: center;
}

.logo {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 0.75rem;
	margin-bottom: 0.5rem;
}

.logo-icon {
	font-size: 2rem;
}

.logo h1 {
	font-size: 1.75rem;
	font-weight: 700;
	margin: 0;
	color: #111827;
}

.subtitle {
	color: #6b7280;
	margin: 0 0 2rem 0;
	font-size: 0.95rem;
}

.error {
	background: #fee2e2;
	color: #991b1b;
	padding: 0.75rem 1rem;
	border-radius: 0.5rem;
	margin-bottom: 1.5rem;
	font-size: 0.875rem;
}

.github-btn {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	gap: 0.5rem;
	width: 100%;
	background: #24292e;
	color: white;
	border: none;
	padding: 0.75rem 1.5rem;
	border-radius: 0.5rem;
	font-size: 1rem;
	font-weight: 600;
	cursor: pointer;
	transition: background 0.15s ease;
}

.github-btn:hover {
	background: #1f2328;
}

.github-btn:active {
	background: #15181c;
}
</style>
