import { derived, writable } from 'svelte/store';
import { api, type BatchAddResult, type Repository } from '$api/client';

// Sort alphabetically, case-insensitive (ABC順ソート)
const sortByName = (repos: Repository[]) =>
	[...repos].sort((a, b) =>
		a.fullName.localeCompare(b.fullName, undefined, { sensitivity: 'base' }),
	);

function createRepositoriesStore() {
	const { subscribe, set, update } = writable<Repository[]>([]);

	return {
		subscribe,
		load: async () => {
			try {
				const repos = await api.repositories.list();
				set(sortByName(repos));
			} catch (error) {
				console.error('Failed to load repositories:', error);
				throw error;
			}
		},
		add: async (owner: string, name: string) => {
			try {
				const repo = await api.repositories.add(owner, name);
				update((repos) => sortByName([...repos, repo]));
				return repo;
			} catch (error) {
				console.error('Failed to add repository:', error);
				throw error;
			}
		},
		remove: async (id: string) => {
			try {
				await api.repositories.delete(id);
				update((repos) => repos.filter((r) => r.id !== id));
			} catch (error) {
				console.error('Failed to remove repository:', error);
				throw error;
			}
		},
		sync: async (id: string, range?: string) => {
			try {
				return await api.repositories.sync(id, range);
			} catch (error) {
				console.error('Failed to sync repository:', error);
				throw error;
			}
		},
		batchAdd: async (repos: { owner: string; name: string }[]): Promise<BatchAddResult[]> => {
			try {
				const results = await api.repositories.batchAdd(repos);
				// Only add successful results to store
				const newRepos = results.filter((r) => r.success && r.repository).map((r) => r.repository!);
				if (newRepos.length > 0) {
					update((current) => sortByName([...current, ...newRepos]));
				}
				return results;
			} catch (error) {
				console.error('Failed to batch add repositories:', error);
				throw error;
			}
		},
	};
}

export const repositories = createRepositoriesStore();

// Multi-repo selection store (empty array = all repos)
export const selectedRepositories = writable<string[]>([]);

// Backward compat: for single-selection contexts (後方互換: 単一選択用)
export const selectedRepository = derived(selectedRepositories, ($sel) =>
	$sel.length === 1 ? $sel[0] : null,
);

// Whether any repositories are selected (including "all")
export const hasSelection = derived(selectedRepositories, (_$sel) => true);

// Currently selected repository objects
export const currentRepositories = derived(
	[repositories, selectedRepositories],
	([$repositories, $selectedRepositories]) => {
		if ($selectedRepositories.length === 0) return $repositories;
		return $repositories.filter((r) => $selectedRepositories.includes(r.id));
	},
);

// Backward compat: single repository object (後方互換: 単一リポジトリ)
export const currentRepository = derived(
	[repositories, selectedRepository],
	([$repositories, $selectedRepository]) => {
		if (!$selectedRepository) return null;
		return $repositories.find((r) => r.id === $selectedRepository) || null;
	},
);
