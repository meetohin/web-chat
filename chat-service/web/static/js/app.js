class ChatApp {
    constructor() {
        this.token = localStorage.getItem('token');
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.notifications = [];
        this.unreadCount = 0;
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

        // Notification panel toggle
        document.getElementById('notificationToggle').addEventListener('click', () => {
            this.toggleNotificationPanel();
        });

        // Mark all notifications as read
        document.getElementById('markAllRead').addEventListener('click', () => {
            this.markAllNotificationsRead();
        });

        // Close notification panel when clicking outside
        document.addEventListener('click', (e) => {
            const panel = document.getElementById('notificationPanel');
            const toggle = document.getElementById('notificationToggle');

            if (!panel.contains(e.target) && !toggle.contains(e.target)) {
                this.hideNotificationPanel();
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
                const data = JSON.parse(event.data);

                // Проверяем, является ли это уведомлением
                if (data.type === 'notification') {
                    this.handleNotification(data.data);
                } else {
                    // Обычное сообщение чата
                    this.displayMessage(data);
                }
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

    handleNotification(notification) {
        // Добавляем уведомление в список
        this.notifications.unshift(notification);

        // Ограничиваем количество уведомлений в памяти
        if (this.notifications.length > 50) {
            this.notifications = this.notifications.slice(0, 50);
        }

        // Увеличиваем счетчик непрочитанных
        this.unreadCount++;
        this.updateNotificationBadge();

        // Обновляем список уведомлений
        this.updateNotificationList();

        // Показываем браузерное уведомление, если разрешено
        this.showBrowserNotification(notification);

        console.log('Received notification:', notification);
    }

    showBrowserNotification(notification) {
        if (Notification.permission === 'granted') {
            const browserNotification = new Notification(notification.title, {
                body: notification.message,
                icon: '/static/favicon.ico' // добавьте фавикон если есть
            });

            // Автоматически закрываем через 5 секунд
            setTimeout(() => {
                browserNotification.close();
            }, 5000);
        } else if (Notification.permission !== 'denied') {
            // Запрашиваем разрешение
            Notification.requestPermission().then(permission => {
                if (permission === 'granted') {
                    this.showBrowserNotification(notification);
                }
            });
        }
    }

    updateNotificationBadge() {
        const badge = document.getElementById('notificationBadge');
        if (this.unreadCount > 0) {
            badge.textContent = this.unreadCount > 99 ? '99+' : this.unreadCount;
            badge.classList.add('show');
        } else {
            badge.classList.remove('show');
        }
    }

    updateNotificationList() {
        const list = document.getElementById('notificationList');

        if (this.notifications.length === 0) {
            list.innerHTML = '<div class="no-notifications">No notifications yet</div>';
            return;
        }

        const notificationsHTML = this.notifications.map(notification => {
            const timeAgo = this.getTimeAgo(new Date(notification.created_at));
            return `
                <div class="notification-item unread" data-id="${notification.id}">
                    <div class="notification-title">${this.escapeHtml(notification.title)}</div>
                    <div class="notification-message">${this.escapeHtml(notification.message)}</div>
                    <div class="notification-time">${timeAgo}</div>
                </div>
            `;
        }).join('');

        list.innerHTML = notificationsHTML;

        // Добавляем обработчики кликов на уведомления
        list.querySelectorAll('.notification-item').forEach(item => {
            item.addEventListener('click', () => {
                this.markNotificationAsRead(item);
            });
        });
    }

    markNotificationAsRead(notificationElement) {
        if (notificationElement.classList.contains('unread')) {
            notificationElement.classList.remove('unread');
            this.unreadCount = Math.max(0, this.unreadCount - 1);
            this.updateNotificationBadge();
        }
    }

    markAllNotificationsRead() {
        const unreadItems = document.querySelectorAll('.notification-item.unread');
        unreadItems.forEach(item => {
            item.classList.remove('unread');
        });

        this.unreadCount = 0;
        this.updateNotificationBadge();
    }

    toggleNotificationPanel() {
        const panel = document.getElementById('notificationPanel');
        panel.classList.toggle('show');

        if (panel.classList.contains('show')) {
            this.updateNotificationList();
        }
    }

    hideNotificationPanel() {
        const panel = document.getElementById('notificationPanel');
        panel.classList.remove('show');
    }

    getTimeAgo(date) {
        const now = new Date();
        const diff = Math.floor((now - date) / 1000);

        if (diff < 60) return 'Just now';
        if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
        if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
        if (diff < 2592000) return `${Math.floor(diff / 86400)}d ago`;

        return date.toLocaleDateString();
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
    // Запрашиваем разрешение на уведомления при загрузке страницы
    if ('Notification' in window && Notification.permission === 'default') {
        Notification.requestPermission();
    }

    new ChatApp();
});