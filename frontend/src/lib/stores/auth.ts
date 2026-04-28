import { writable } from 'svelte/store';

const API_BASE = import.meta.env.VITE_API_BASE || '/api';

export interface AuthUser {
	id: string;
	login: string;
	name: string;
	avatarUrl: string;
}

export const authUser = writable<AuthUser | null>(null);
export const authChecked = writable(false);

// checkAuth queries the backend auth endpoint to determine the current login state.
// Resolves with the user (if logged in) or null. Always sets authChecked=true.
export async function checkAuth(): Promise<AuthUser | null> {
	try {
		const res = await fetch(`${API_BASE}/auth/me`, { credentials: 'include' });
		if (res.ok) {
			const user: AuthUser = await res.json();
			authUser.set(user);
			authChecked.set(true);
			return user;
		}
	} catch {
		// network error: treat as logged out
	}
	authUser.set(null);
	authChecked.set(true);
	return null;
}

// redirectToGitHubLogin navigates the browser to the backend OAuth start URL,
// passing the current path so the user returns there after login.
export function redirectToGitHubLogin(returnTo?: string): void {
	const target = returnTo ?? window.location.pathname;
	const params = new URLSearchParams({ return_to: target });
	window.location.href = `${API_BASE}/auth/github/login?${params.toString()}`;
}

export async function logout(): Promise<void> {
	await fetch(`${API_BASE}/auth/logout`, { method: 'POST', credentials: 'include' });
	authUser.set(null);
}
