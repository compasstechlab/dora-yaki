import { writable } from 'svelte/store';

export interface FlashMessage {
	id: number;
	type: 'error' | 'warning' | 'success' | 'info';
	message: string;
}

let nextId = 0;

// Flash message store
export const flashMessages = writable<FlashMessage[]>([]);

// Add message with auto-dismiss (メッセージ追加・自動消去付き)
export function addFlash(type: FlashMessage['type'], message: string, durationMs = 5000) {
	const id = nextId++;
	flashMessages.update((msgs) => [...msgs, { id, type, message }]);

	if (durationMs > 0) {
		setTimeout(() => removeFlash(id), durationMs);
	}
}

// Remove message (メッセージ削除)
export function removeFlash(id: number) {
	flashMessages.update((msgs) => msgs.filter((m) => m.id !== id));
}
