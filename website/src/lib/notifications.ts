import { writable } from 'svelte/store';

export type NotificationType = 'info' | 'success' | 'error' | 'warning';

export interface Notification {
	id: string;
	type: NotificationType;
	message: string;
	timeout?: number;
}

function createNotificationStore() {
	const { subscribe, update } = writable<Notification[]>([]);

	function add(message: string, type: NotificationType = 'info', timeout = 5000) {
		const id = Math.random().toString(36).substring(2, 9);
		update(n => [...n, { id, type, message, timeout }]);

		if (timeout > 0) {
			setTimeout(() => {
				remove(id);
			}, timeout);
		}
		return id;
	}

	function remove(id: string) {
		update(n => n.filter(item => item.id !== id));
	}

	return {
		subscribe,
		info: (msg: string, timeout?: number) => add(msg, 'info', timeout),
		success: (msg: string, timeout?: number) => add(msg, 'success', timeout),
		error: (msg: string, timeout?: number) => add(msg, 'error', timeout),
		warning: (msg: string, timeout?: number) => add(msg, 'warning', timeout),
		remove
	};
}

export const notifications = createNotificationStore();
