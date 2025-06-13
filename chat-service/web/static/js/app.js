class ChatApp {
    constructor() {
        this.token = localStorage.getItem('token');
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.init();
    }

    init() {
        if (!this.token) {
            window.location.href = '/login';
            return;
        }

        this.setupEventListeners();
        this.connectWebSocket();
        this.loadStats();
    }

    setupEventListeners() {
        document.getElementById('messageForm').addEventListener('submit', (e) => {
            e.preventDefault();
            this.sendMessage();
        });

        document.getElementById('logout').addEventListener('click', () => {
            this.logout();
        });

        // Enter key to send message
        document.getElementById('messageInput').addEventListener('keypress', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.sendMessage();
            }
        });
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws?token=${this.token}`;

        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectAttempts = 0;
            this.updateConnectionStatus(true);
        };

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data);
                this.displayMessage(message);
            } catch (error) {
                console.error('Failed to parse message:', error);
            }
        };

        this.ws.onclose = (event) => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus(false);

            if (event.code === 1006 || event.code === 1000) {
                this.handleReconnect();
            }
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.updateConnectionStatus(false);
        };
    }

    handleReconnect() {
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++;
            console.log(`Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);

            setTimeout(() => {
                this.connectWebSocket();
            }, this.reconnectDelay * this.reconnectAttempts);
        } else {
            console.error('Max reconnection attempts reached');
            this.showError('Connection lost. Please refresh the page.');
        }
    }

    sendMessage() {
        const input = document.getElementById('messageInput');
        const text = input.value.trim();

        if (text && this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ text }));
            input.value = '';
        } else if (!text) {
            this.showError('Please enter a message');
        } else {
            this.showError('Connection lost. Trying to reconnect...');
        }
    }

    displayMessage(message) {
        const messagesContainer = document.getElementById('messages');

        // Remove welcome message if exists
        const welcomeMessage = messagesContainer.querySelector('.welcome-message');
        if (welcomeMessage) {
            welcomeMessage.remove();
        }

        const messageElement = document.createElement('div');
        messageElement.className = 'message';

        const timestamp = new Date(message.timestamp).toLocaleTimeString();

        messageElement.innerHTML = `
            <div class="message-header">
                <span class="message-username">${this.escapeHtml(message.username)}</span>
                <span class="message-time">${timestamp}</span>
            </div>
            <div class="message-text">${this.escapeHtml(message.text)}</div>
        `;

        messagesContainer.appendChild(messageElement);
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    updateConnectionStatus(connected) {
        const header = document.querySelector('.chat-header');

        if (connected) {
            header.style.background = 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)';
        } else {
            header.style.background = 'linear-gradient(135deg, #dc3545 0%, #c82333 100%)';
        }
    }

    async loadStats() {
        try {
            const response = await fetch('/api/stats');
            if (response.ok) {
                const stats = await response.json();
                document.getElementById('userCount').textContent =
                    `${stats.connected_clients} users online`;
            }
        } catch (error) {
            console.error('Failed to load stats:', error);
        }

        // Update stats every 30 seconds
        setTimeout(() => this.loadStats(), 30000);
    }

    showError(message) {
        // Create a temporary error message
        const errorDiv = document.createElement('div');
        errorDiv.className = 'error';
        errorDiv.textContent = message;
        errorDiv.style.position = 'fixed';
        errorDiv.style.top = '20px';
        errorDiv.style.right = '20px';
        errorDiv.style.zIndex = '1000';
        errorDiv.style.display = 'block';

        document.body.appendChild(errorDiv);

        setTimeout(() => {
            errorDiv.remove();
        }, 5000);
    }

    logout() {
        localStorage.removeItem('token');
        if (this.ws) {
            this.ws.close();
        }
        window.location.href = '/login';
    }
}

// Initialize the chat application when the page loads
document.addEventListener('DOMContentLoaded', () => {
    new ChatApp();
});