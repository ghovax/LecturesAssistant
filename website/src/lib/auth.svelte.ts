import { api } from './api/client';

class AuthStore {
    user = $state<any>(null);
    initialized = $state(false);
    loading = $state(true);

    async check() {
        this.loading = true;
        try {
            const status = await api.getStatus();
            this.user = status.authenticated ? status.user : null;
            this.initialized = status.initialized;
        } catch (e) {
            this.user = null;
        } finally {
            this.loading = false;
        }
    }

    async logout() {
        await api.logout();
        this.user = null;
    }
}

export const auth = new AuthStore();
