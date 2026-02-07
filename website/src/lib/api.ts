export async function apiFetch(url: string, options: RequestInit = {}) {
	const headers = {
		'X-Requested-With': 'XMLHttpRequest',
		...options.headers
	};

	if (options.body && !(options.body instanceof FormData) && typeof options.body === 'object') {
		(headers as any)['Content-Type'] = 'application/json';
		options.body = JSON.stringify(options.body);
	}

	const response = await fetch(url, { ...options, headers });
	const json = await response.json();

	if (!response.ok) {
		throw new Error(json.error?.message || json.message || 'API request failed');
	}

	return json.data;
}

export const auth = {
	async getStatus() {
		return apiFetch('/api/auth/status');
	},
	async login(credentials: any) {
		return apiFetch('/api/auth/login', { method: 'POST', body: credentials });
	},
	async setup(credentials: any) {
		return apiFetch('/api/auth/setup', { method: 'POST', body: credentials });
	},
	async logout() {
		return apiFetch('/api/auth/logout', { method: 'POST' });
	}
};
