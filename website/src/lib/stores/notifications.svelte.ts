export interface Notification {
    id: string;
    message: string;
    type: 'success' | 'error' | 'info';
    timeout?: number;
}

class NotificationStore {
    notifications = $state<Notification[]>([]);

    add(message: string, type: Notification['type'] = 'info', timeout = 5000) {
        const id = Math.random().toString(36).substring(2, 9);
        const notification: Notification = { id, message, type, timeout };
        this.notifications.push(notification);

        if (timeout > 0) {
            setTimeout(() => {
                this.remove(id);
            }, timeout);
        }
        return id;
    }

    success(message: string, timeout = 5000) {
        return this.add(message, 'success', timeout);
    }

    error(message: string, timeout = 8000) {
        return this.add(message, 'error', timeout);
    }

    info(message: string, timeout = 5000) {
        return this.add(message, 'info', timeout);
    }

    remove(id: string) {
        this.notifications = this.notifications.filter(n => n.id !== id);
    }
}

export const notifications = new NotificationStore();
