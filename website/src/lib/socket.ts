import { browser } from '$app/environment';

type MessageHandler = (payload: any) => void;

class SocketManager {
	private socket: WebSocket | null = null;
	private handlers: Map<string, Set<MessageHandler>> = new Map();
	private subscriptions: Set<string> = new Set();
	private reconnectTimeout: any = null;

	constructor() {
		if (browser) {
			this.connect();
		}
	}

	private connect() {
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const host = window.location.host;
		this.socket = new WebSocket(`${protocol}//${host}/api/socket`);

		this.socket.onopen = () => {
			console.log('[WS] Connected');
			// Resubscribe to existing channels
			for (const channel of this.subscriptions) {
				this.send({ type: 'subscribe', channel });
			}
		};

		this.socket.onmessage = (event) => {
			try {
				const message = JSON.parse(event.data);
				const channelHandlers = this.handlers.get(message.channel);
				if (channelHandlers) {
					for (const handler of channelHandlers) {
						handler(message);
					}
				}
			} catch (e) {
				console.error('[WS] Error parsing message', e);
			}
		};

		this.socket.onclose = () => {
			console.log('[WS] Disconnected, reconnecting...');
			this.reconnectTimeout = setTimeout(() => this.connect(), 3000);
		};
	}

	private send(data: any) {
		if (this.socket && this.socket.readyState === WebSocket.OPEN) {
			this.socket.send(JSON.stringify(data));
		}
	}

	subscribe(channel: string, handler: MessageHandler) {
		if (!this.handlers.has(channel)) {
			this.handlers.set(channel, new Set());
		}
		this.handlers.get(channel)!.add(handler);
		
		if (!this.subscriptions.has(channel)) {
			this.subscriptions.add(channel);
			this.send({ type: 'subscribe', channel });
		}

		return () => this.unsubscribe(channel, handler);
	}

	unsubscribe(channel: string, handler: MessageHandler) {
		const channelHandlers = this.handlers.get(channel);
		if (channelHandlers) {
			channelHandlers.delete(handler);
			if (channelHandlers.size === 0) {
				this.handlers.delete(channel);
				this.subscriptions.delete(channel);
				this.send({ type: 'unsubscribe', channel });
			}
		}
	}
}

export const socketManager = new SocketManager();
