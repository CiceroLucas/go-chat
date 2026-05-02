

(() => {
    'use strict';

    const state = {
        token: localStorage.getItem('go_chat_token') || null,
        username: localStorage.getItem('go_chat_username') || null,
        userId: localStorage.getItem('go_chat_user_id') || null,
        currentRoom: null,
        currentRoomName: null,
        ws: null,
        reconnectAttempts: 0,
        maxReconnectAttempts: 5,
        reconnectDelay: 1000,
        rooms: [],
    };

    const $ = (sel) => document.querySelector(sel);
    const $$ = (sel) => document.querySelectorAll(sel);

    const dom = {

        authScreen: $('#auth-screen'),
        chatScreen: $('#chat-screen'),

        loginForm: $('#login-form'),
        registerForm: $('#register-form'),
        showRegister: $('#show-register'),
        showLogin: $('#show-login'),
        authError: $('#auth-error'),

        sidebar: $('#sidebar'),
        toggleSidebar: $('#toggle-sidebar'),
        userName: $('#user-name'),
        userAvatar: $('#user-avatar'),
        roomList: $('#room-list'),
        userList: $('#user-list'),
        onlineCount: $('#online-count'),
        currentRoomName: $('#current-room-name'),
        currentRoomStatus: $('#current-room-status'),
        connectionStatus: $('#connection-status'),
        messagesContainer: $('#messages-container'),
        messagesEmpty: $('#messages-empty'),
        messages: $('#messages'),
        messageInputContainer: $('#message-input-container'),
        messageForm: $('#message-form'),
        messageInput: $('#message-input'),
        logoutBtn: $('#logout-btn'),

        createRoomBtn: $('#create-room-btn'),
        createRoomModal: $('#create-room-modal'),
        createRoomForm: $('#create-room-form'),
        cancelRoomBtn: $('#cancel-room-btn'),
        roomNameInput: $('#room-name-input'),

        toastContainer: $('#toast-container'),
    };

    const API_BASE = `${window.location.origin}/api`;

    async function api(endpoint, options = {}) {
        const url = `${API_BASE}${endpoint}`;
        const headers = { 'Content-Type': 'application/json' };

        if (state.token) {
            headers['Authorization'] = `Bearer ${state.token}`;
        }

        const response = await fetch(url, {
            ...options,
            headers: { ...headers, ...options.headers },
        });

        const contentType = response.headers.get('content-type');
        if (!contentType || !contentType.includes('application/json')) {
            const text = await response.text();
            throw new Error(`Erro no servidor (${response.status}): ${text.substring(0, 50)}`);
        }

        const data = await response.json();

        if (!data.success) {
            throw new Error(data.error || 'Erro desconhecido');
        }

        return data.data;
    }

    function showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        dom.toastContainer.appendChild(toast);

        setTimeout(() => toast.remove(), 3500);
    }

    function showAuthError(msg) {
        dom.authError.textContent = msg;
        dom.authError.style.display = 'block';
        setTimeout(() => { dom.authError.style.display = 'none'; }, 5000);
    }

    dom.showRegister.addEventListener('click', (e) => {
        e.preventDefault();
        dom.loginForm.classList.remove('active');
        dom.registerForm.classList.add('active');
        dom.authError.style.display = 'none';
    });

    dom.showLogin.addEventListener('click', (e) => {
        e.preventDefault();
        dom.registerForm.classList.remove('active');
        dom.loginForm.classList.add('active');
        dom.authError.style.display = 'none';
    });

    dom.loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = $('#login-username').value.trim();
        const password = $('#login-password').value;

        if (!username || !password) return;

        const btn = $('#login-btn');
        btn.querySelector('.btn-text').style.display = 'none';
        btn.querySelector('.btn-loader').style.display = 'inline';
        btn.disabled = true;

        try {
            const result = await api('/auth/login', {
                method: 'POST',
                body: JSON.stringify({ username, password }),
            });

            saveAuth(result.token, result.user);
            showToast('Login realizado com sucesso!', 'success');
            enterChat();
        } catch (err) {
            showAuthError(err.message);
        } finally {
            btn.querySelector('.btn-text').style.display = 'inline';
            btn.querySelector('.btn-loader').style.display = 'none';
            btn.disabled = false;
        }
    });

    dom.registerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = $('#register-username').value.trim();
        const email = $('#register-email').value.trim();
        const password = $('#register-password').value;

        if (!username || !email || !password) return;

        const btn = $('#register-btn');
        btn.querySelector('.btn-text').style.display = 'none';
        btn.querySelector('.btn-loader').style.display = 'inline';
        btn.disabled = true;

        try {
            const result = await api('/auth/register', {
                method: 'POST',
                body: JSON.stringify({ username, email, password }),
            });

            saveAuth(result.token, result.user);
            showToast('Conta criada com sucesso!', 'success');
            enterChat();
        } catch (err) {
            showAuthError(err.message);
        } finally {
            btn.querySelector('.btn-text').style.display = 'inline';
            btn.querySelector('.btn-loader').style.display = 'none';
            btn.disabled = false;
        }
    });

    function saveAuth(token, user) {
        state.token = token;
        state.username = user.username;
        state.userId = user.id;
        localStorage.setItem('go_chat_token', token);
        localStorage.setItem('go_chat_username', user.username);
        localStorage.setItem('go_chat_user_id', user.id);
    }

    function clearAuth() {
        state.token = null;
        state.username = null;
        state.userId = null;
        state.currentRoom = null;
        localStorage.removeItem('go_chat_token');
        localStorage.removeItem('go_chat_username');
        localStorage.removeItem('go_chat_user_id');
    }

    dom.logoutBtn.addEventListener('click', () => {
        if (state.ws) state.ws.close();
        clearAuth();
        dom.chatScreen.classList.remove('active');
        dom.authScreen.classList.add('active');
        showToast('Até mais!', 'info');
    });

    async function enterChat() {
        dom.authScreen.classList.remove('active');
        dom.chatScreen.classList.add('active');

        dom.userName.textContent = state.username;
        dom.userAvatar.textContent = state.username.charAt(0).toUpperCase();

        await loadRooms();

        connectWebSocket();
    }

    async function loadRooms() {
        try {
            const rooms = await api('/rooms');
            state.rooms = rooms || [];
            renderRooms();
        } catch (err) {
            console.error('Erro ao carregar salas:', err);
            showToast('Erro ao carregar salas', 'error');
        }
    }

    function renderRooms() {
        dom.roomList.innerHTML = '';
        state.rooms.forEach(room => {
            const li = document.createElement('li');

            const nameSpan = document.createElement('span');
            nameSpan.className = 'room-name';
            nameSpan.textContent = room.name;
            li.appendChild(nameSpan);

            li.dataset.roomId = room.id;
            if (room.id === state.currentRoom) {
                li.classList.add('active');
            }
            nameSpan.addEventListener('click', () => joinRoom(room.id, room.name));

            if (room.name !== 'Geral') {
                const deleteBtn = document.createElement('button');
                deleteBtn.className = 'btn-room-delete';
                deleteBtn.textContent = 'x';
                deleteBtn.title = 'Deletar sala';
                deleteBtn.addEventListener('click', (e) => {
                    e.stopPropagation();
                    deleteRoom(room.id, room.name);
                });
                li.appendChild(deleteBtn);
            }

            dom.roomList.appendChild(li);
        });
    }

    dom.createRoomBtn.addEventListener('click', () => {
        dom.createRoomModal.style.display = 'flex';
        dom.roomNameInput.focus();
    });

    dom.cancelRoomBtn.addEventListener('click', () => {
        dom.createRoomModal.style.display = 'none';
        dom.roomNameInput.value = '';
    });

    dom.createRoomModal.querySelector('.modal-backdrop').addEventListener('click', () => {
        dom.createRoomModal.style.display = 'none';
        dom.roomNameInput.value = '';
    });

    dom.createRoomForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const name = dom.roomNameInput.value.trim();
        if (!name) return;

        try {
            await api('/rooms', {
                method: 'POST',
                body: JSON.stringify({ name }),
            });

            dom.createRoomModal.style.display = 'none';
            dom.roomNameInput.value = '';
            showToast(`Sala "${name}" criada!`, 'success');
            await loadRooms();
        } catch (err) {
            showToast(err.message, 'error');
        }
    });

    async function deleteRoom(roomId, roomName) {
        if (!confirm(`Deletar a sala "${roomName}"? Todas as mensagens serao perdidas.`)) {
            return;
        }

        try {
            await api(`/rooms/${roomId}`, { method: 'DELETE' });
            showToast(`Sala "${roomName}" deletada`, 'success');

            if (state.currentRoom === roomId) {
                state.currentRoom = null;
                state.currentRoomName = null;
                dom.currentRoomName.textContent = 'Selecione uma sala';
                dom.currentRoomStatus.textContent = '';
                dom.messages.innerHTML = '';
                dom.messagesEmpty.style.display = 'flex';
                dom.messageInputContainer.style.display = 'none';
                dom.userList.innerHTML = '';
                dom.onlineCount.textContent = '(0)';
            }

            await loadRooms();
        } catch (err) {
            showToast(err.message, 'error');
        }
    }

    function joinRoom(roomId, roomName) {
        if (state.currentRoom === roomId) return;

        if (state.currentRoom && state.ws) {
            sendWS({ type: 'leave_room', room_id: state.currentRoom });
        }

        state.currentRoom = roomId;
        state.currentRoomName = roomName;

        dom.currentRoomName.textContent = `# ${roomName}`;
        dom.currentRoomStatus.textContent = '';
        dom.messagesEmpty.style.display = 'none';
        dom.messages.innerHTML = '';
        dom.messageInputContainer.style.display = 'block';
        dom.messageInput.focus();

        dom.roomList.querySelectorAll('li').forEach(li => {
            li.classList.toggle('active', li.dataset.roomId === roomId);
        });

        if (state.ws && state.ws.readyState === WebSocket.OPEN) {
            sendWS({ type: 'join_room', room_id: roomId });

            setTimeout(() => {
                sendWS({ type: 'room_history', room_id: roomId });
            }, 200);
        }

        dom.sidebar.classList.remove('open');
    }

    function connectWebSocket() {
        if (state.ws && state.ws.readyState === WebSocket.OPEN) {
            state.ws.close();
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws?token=${state.token}`;

        state.ws = new WebSocket(wsUrl);

        state.ws.onopen = () => {
            console.log('WebSocket conectado');
            state.reconnectAttempts = 0;
            updateConnectionStatus(true);

            if (state.currentRoom) {
                sendWS({ type: 'join_room', room_id: state.currentRoom });
                setTimeout(() => {
                    sendWS({ type: 'room_history', room_id: state.currentRoom });
                }, 200);
            }
        };

        state.ws.onmessage = (event) => {

            const parts = event.data.split('\n');
            parts.forEach(part => {
                if (!part.trim()) return;
                try {
                    const msg = JSON.parse(part);
                    handleWSMessage(msg);
                } catch (err) {
                    console.error('Erro ao parsear mensagem:', err);
                }
            });
        };

        state.ws.onclose = (event) => {
            console.log('WebSocket desconectado:', event.code);
            updateConnectionStatus(false);

            if (state.token && state.reconnectAttempts < state.maxReconnectAttempts) {
                state.reconnectAttempts++;
                const delay = state.reconnectDelay * Math.pow(2, state.reconnectAttempts - 1);
                console.log(`Reconectando em ${delay}ms (tentativa ${state.reconnectAttempts})...`);
                setTimeout(connectWebSocket, delay);
            }
        };

        state.ws.onerror = (err) => {
            console.error('Erro no WebSocket:', err);
        };
    }

    function sendWS(data) {
        if (state.ws && state.ws.readyState === WebSocket.OPEN) {
            state.ws.send(JSON.stringify(data));
        }
    }

    function updateConnectionStatus(connected) {
        const el = dom.connectionStatus;
        if (connected) {
            el.classList.add('connected');
            el.querySelector('.status-text').textContent = 'Conectado';
        } else {
            el.classList.remove('connected');
            el.querySelector('.status-text').textContent = 'Desconectado';
        }
    }

    function handleWSMessage(msg) {
        switch (msg.type) {
            case 'message':
                appendMessage(msg);
                break;

            case 'system':
                appendSystemMessage(msg);
                break;

            case 'room_history':
                renderHistory(msg);
                break;

            case 'user_list':
                renderUserList(msg);
                break;

            case 'error':
                showToast(msg.content, 'error');
                break;

            default:
                console.log('Mensagem desconhecida:', msg);
        }
    }

    function appendMessage(msg) {
        if (msg.room_id !== state.currentRoom) return;

        const div = document.createElement('div');
        const isOwn = msg.username === state.username;
        div.className = `message ${isOwn ? 'own' : ''}`;

        const initial = (msg.username || '?').charAt(0).toUpperCase();
        const time = formatTime(msg.timestamp);

        div.innerHTML = `
            <div class="message-avatar">${initial}</div>
            <div class="message-body">
                <div class="message-header">
                    <span class="message-username">${escapeHTML(msg.username)}</span>
                    <span class="message-time">${time}</span>
                </div>
                <div class="message-content">${escapeHTML(msg.content)}</div>
            </div>
        `;

        dom.messages.appendChild(div);
        scrollToBottom();
    }

    function appendSystemMessage(msg) {
        if (msg.room_id !== state.currentRoom) return;

        const div = document.createElement('div');
        div.className = 'message system';
        div.innerHTML = `
            <div class="message-content">— ${escapeHTML(msg.content)} —</div>
        `;
        dom.messages.appendChild(div);
        scrollToBottom();
    }

    function renderHistory(msg) {
        if (msg.room_id !== state.currentRoom) return;

        let messages = [];
        try {
            messages = (typeof msg.data === 'string' ? JSON.parse(msg.data) : msg.data) || [];
        } catch {
            return;
        }

        dom.messages.innerHTML = '';

        if (messages.length === 0) {
            const div = document.createElement('div');
            div.className = 'message system';
            div.innerHTML = `<div class="message-content">— Nenhuma mensagem anterior —</div>`;
            dom.messages.appendChild(div);
            return;
        }

        messages.forEach(m => {
            const div = document.createElement('div');
            const isOwn = m.username === state.username;
            const isSystem = m.type === 'system';

            if (isSystem) {
                div.className = 'message system';
                div.innerHTML = `<div class="message-content">— ${escapeHTML(m.content)} —</div>`;
            } else {
                div.className = `message ${isOwn ? 'own' : ''}`;
                const initial = (m.username || '?').charAt(0).toUpperCase();
                const time = formatTime(m.created_at);
                div.innerHTML = `
                    <div class="message-avatar">${initial}</div>
                    <div class="message-body">
                        <div class="message-header">
                            <span class="message-username">${escapeHTML(m.username)}</span>
                            <span class="message-time">${time}</span>
                        </div>
                        <div class="message-content">${escapeHTML(m.content)}</div>
                    </div>
                `;
            }
            dom.messages.appendChild(div);
        });

        scrollToBottom();
    }

    function renderUserList(msg) {
        if (msg.room_id !== state.currentRoom) return;

        let users = [];
        try {
            users = (typeof msg.data === 'string' ? JSON.parse(msg.data) : msg.data) || [];
        } catch {
            return;
        }

        dom.userList.innerHTML = '';
        dom.onlineCount.textContent = `(${users.length})`;
        dom.currentRoomStatus.textContent = `${users.length} online`;

        users.forEach(username => {
            const li = document.createElement('li');
            li.textContent = username;
            dom.userList.appendChild(li);
        });
    }

    dom.messageForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const content = dom.messageInput.value.trim();
        if (!content || !state.currentRoom) return;

        sendWS({
            type: 'message',
            room_id: state.currentRoom,
            content: content,
        });

        dom.messageInput.value = '';
        dom.messageInput.focus();
    });

    dom.toggleSidebar.addEventListener('click', () => {
        dom.sidebar.classList.toggle('open');
    });

    function formatTime(timestamp) {
        if (!timestamp) return '';
        const date = new Date(timestamp);
        if (isNaN(date.getTime())) return '';
        return date.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });
    }

    function escapeHTML(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    function scrollToBottom() {
        dom.messagesContainer.scrollTop = dom.messagesContainer.scrollHeight;
    }

    if (state.token && state.username) {
        enterChat();
    }

})();
