import { nanoid } from "nanoid";

export interface Notification {
  id: string;
  message: string;
  type: "success" | "error" | "info";
  timeout?: number;
  action?: {
    label: string;
    callback: () => void;
  };
}

class NotificationStore {
  notifications = $state<Notification[]>([]);

  add(
    message: string,
    type: Notification["type"] = "info",
    timeout = 5000,
    action?: Notification["action"],
  ) {
    const id = nanoid();
    const notification: Notification = { id, message, type, timeout, action };
    this.notifications.push(notification);

    if (timeout > 0) {
      setTimeout(() => {
        this.remove(id);
      }, timeout);
    }
    return id;
  }

  success(message: string, timeout = 5000, action?: Notification["action"]) {
    return this.add(message, "success", timeout, action);
  }

  error(message: string, timeout = 8000, action?: Notification["action"]) {
    return this.add(message, "error", timeout, action);
  }

  info(message: string, timeout = 5000, action?: Notification["action"]) {
    return this.add(message, "info", timeout, action);
  }

  remove(id: string) {
    this.notifications = this.notifications.filter((n) => n.id !== id);
  }
}

export const notifications = new NotificationStore();
