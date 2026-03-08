/* ═══════════════════════════════════════════════
   DevConnect — WebSocket Chat Client
   Real-time messaging with reconnect logic
   ═══════════════════════════════════════════════ */

let ws = null;
let chatUsername = '';
let currentRoomId = 'room_general';
let reconnectTimer = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_DELAY = 3000;

// ─── Join Chat ───
function joinChat() {
    const input = document.getElementById('chat-username');
    const username = input.value.trim();
    if (!username) {
        input.style.borderColor = 'var(--red)';
        setTimeout(() => input.style.borderColor = '', 2000);
        return;
    }

    chatUsername = username;
    document.getElementById('chat-login').style.display = 'none';
    document.getElementById('chat-interface').style.display = 'grid';

    connectWebSocket();
    loadRoomMessages(currentRoomId);
}

// ─── Load Rooms ───
async function loadRooms() {
    try {
        const res = await fetch('/api/v1/rooms');
        const rooms = await res.json();
        renderRoomList(rooms);
    } catch (err) {
        console.error('Failed to load rooms:', err);
    }
}

function renderRoomList(rooms) {
    const list = document.getElementById('room-list');
    if (!list) return;

    list.innerHTML = rooms.map(r => `
        <div class="room-item ${r.id === currentRoomId ? 'active' : ''}" 
             onclick="switchRoom('${r.id}', '${r.name}')" 
             id="room-btn-${r.id}">
            <div class="room-item-name">${r.name}</div>
            <div class="room-item-desc">${r.description}</div>
        </div>
    `).join('');
}

// ─── Switch Room ───
function switchRoom(roomId, roomName) {
    if (roomId === currentRoomId) return;

    // Disconnect from current room
    if (ws) {
        ws.close();
    }

    currentRoomId = roomId;

    // Update UI
    document.querySelectorAll('.room-item').forEach(r => r.classList.remove('active'));
    const roomBtn = document.getElementById(`room-btn-${roomId}`);
    if (roomBtn) roomBtn.classList.add('active');

    document.getElementById('current-room-name').textContent = roomName;

    // Clear messages and load history
    document.getElementById('chat-messages').innerHTML = '';
    loadRoomMessages(roomId);

    // Reconnect to new room
    connectWebSocket();
}

// ─── Load Room Messages ───
async function loadRoomMessages(roomId) {
    try {
        const res = await fetch(`/api/v1/rooms/${roomId}/messages`);
        const messages = await res.json();

        const container = document.getElementById('chat-messages');
        messages.forEach(msg => {
            appendChatMessage(msg.username, msg.content, msg.timestamp, 'chat');
        });
        container.scrollTop = container.scrollHeight;
    } catch (err) {
        console.error('Failed to load messages:', err);
    }
}

// ─── WebSocket Connection ───
function connectWebSocket() {
    if (ws) {
        ws.close();
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws?username=${encodeURIComponent(chatUsername)}&room=${encodeURIComponent(currentRoomId)}`;

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log('✅ WebSocket connected');
        reconnectAttempts = 0;
        updateRoomOnlineCount();
    };

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);

            if (msg.type === 'system') {
                appendSystemMessage(msg.content);
            } else if (msg.type === 'chat') {
                appendChatMessage(msg.username, msg.content, new Date().toISOString(), 'chat');
            }

            // Auto-scroll
            const container = document.getElementById('chat-messages');
            container.scrollTop = container.scrollHeight;
        } catch (err) {
            console.error('Failed to parse WebSocket message:', err);
        }
    };

    ws.onclose = () => {
        console.log('❌ WebSocket disconnected');
        attemptReconnect();
    };

    ws.onerror = (err) => {
        console.error('WebSocket error:', err);
    };
}

function attemptReconnect() {
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
        appendSystemMessage('Connection lost. Please refresh the page.');
        return;
    }

    reconnectAttempts++;
    console.log(`🔄 Reconnecting... attempt ${reconnectAttempts}`);

    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(() => {
        if (chatUsername) connectWebSocket();
    }, RECONNECT_DELAY);
}

// ─── Send Message ───
function sendMessage() {
    const input = document.getElementById('chat-input');
    const content = input.value.trim();
    if (!content || !ws || ws.readyState !== WebSocket.OPEN) return;

    const msg = {
        type: 'chat',
        content: content,
        username: chatUsername,
        room_id: currentRoomId,
    };

    ws.send(JSON.stringify(msg));
    input.value = '';
    input.focus();
}

function handleChatKeypress(e) {
    if (e.key === 'Enter') sendMessage();
}

// ─── Render Messages ───
function appendChatMessage(username, content, timestamp, type) {
    const container = document.getElementById('chat-messages');
    if (!container) return;

    const time = timestamp ? new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '';
    const initials = username.slice(0, 2).toUpperCase();
    const isMe = username === chatUsername;

    const msgEl = document.createElement('div');
    msgEl.className = 'chat-msg';
    msgEl.innerHTML = `
        <div class="chat-msg-avatar" style="background:${isMe ? 'linear-gradient(135deg, #10b981, #059669)' : 'var(--gradient-primary)'}">
            ${initials}
        </div>
        <div class="chat-msg-body">
            <div class="chat-msg-header">
                <span class="chat-msg-name" style="${isMe ? 'color:var(--green)' : ''}">${escapeHtmlChat(username)}</span>
                <span class="chat-msg-time">${time}</span>
            </div>
            <div class="chat-msg-text">${escapeHtmlChat(content)}</div>
        </div>
    `;
    container.appendChild(msgEl);
}

function appendSystemMessage(content) {
    const container = document.getElementById('chat-messages');
    if (!container) return;

    const msgEl = document.createElement('div');
    msgEl.className = 'chat-msg-system';
    msgEl.textContent = content;
    container.appendChild(msgEl);
}

function escapeHtmlChat(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ─── Update Online Count ───
async function updateRoomOnlineCount() {
    try {
        const res = await fetch('/api/v1/stats');
        const data = await res.json();
        const onlineEl = document.getElementById('room-online');
        if (onlineEl) onlineEl.textContent = `${data.online_now} online`;

        const headerOnline = document.getElementById('online-num');
        if (headerOnline) headerOnline.textContent = data.online_now;
    } catch (err) {
        // Ignore
    }
}

// Refresh online count periodically
setInterval(updateRoomOnlineCount, 10000);
