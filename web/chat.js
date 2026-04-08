/* ═══════════════════════════════════════════════
   DevConnect — DM-Only Chat Client
   Personal 1-on-1 messaging between developers
   No general/topic rooms — DM only
   Uses JWT auth via authFetch() from app.js
   ═══════════════════════════════════════════════ */

let ws = null;
let currentRoomId = null;
let currentChatPartner = null;
let reconnectTimer = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_DELAY = 3000;

// ─── Load People List for DM ───
async function loadPeopleForDM() {
    if (!currentUser) return;

    try {
        const res = await authFetch('/api/v1/users');
        const users = await res.json();

        // Filter out current user — you can't DM yourself
        const otherUsers = users.filter(u => u.id !== currentUser.id);
        renderPeopleList(otherUsers);
    } catch (err) {
        console.error('Failed to load people for DM:', err);
    }
}

function renderPeopleList(users) {
    const list = document.getElementById('dm-people-list');
    if (!list) return;

    list.innerHTML = users.map(u => `
        <div class="room-item dm-person-item" 
             onclick="selectDMPerson('${u.id}', '${escapeHtmlChat(u.username)}')"
             id="dm-person-${u.id}">
            <div class="dm-avatar-mini">
                <img src="${u.avatar_url}" alt="${u.username}">
            </div>
            <div class="dm-person-info">
                <div class="room-item-name">${u.username}</div>
                <div class="dm-person-skills">${u.skills.slice(0, 2).join(', ')}</div>
            </div>
        </div>
    `).join('');

    // Wire up search
    const searchInput = document.getElementById('people-search');
    if (searchInput) {
        searchInput.oninput = (e) => {
            const q = e.target.value.toLowerCase();
            const filtered = users.filter(u =>
                u.username.toLowerCase().includes(q) ||
                u.skills.some(s => s.toLowerCase().includes(q)) ||
                (u.location && u.location.toLowerCase().includes(q))
            );
            renderFilteredPeople(filtered);
        };
    }
}

function renderFilteredPeople(users) {
    const list = document.getElementById('dm-people-list');
    if (!list) return;

    list.innerHTML = users.map(u => `
        <div class="room-item dm-person-item ${currentChatPartner === u.id ? 'active' : ''}" 
             onclick="selectDMPerson('${u.id}', '${escapeHtmlChat(u.username)}')"
             id="dm-person-${u.id}">
            <div class="dm-avatar-mini">
                <img src="${u.avatar_url}" alt="${u.username}">
            </div>
            <div class="dm-person-info">
                <div class="room-item-name">${u.username}</div>
                <div class="dm-person-skills">${u.skills.slice(0, 2).join(', ')}</div>
            </div>
        </div>
    `).join('');
}

// ─── Select a person and open DM ───
async function selectDMPerson(userId, username, existingRoomId) {
    if (!currentUser) return;

    currentChatPartner = userId;

    // Highlight in sidebar
    document.querySelectorAll('.dm-person-item').forEach(r => r.classList.remove('active'));
    const el = document.getElementById(`dm-person-${userId}`);
    if (el) el.classList.add('active');

    // Update chat header
    const headerArea = document.getElementById('chat-header-area');
    headerArea.innerHTML = `
        <div class="dm-chat-header">
            <img src="https://api.dicebear.com/7.x/avataaars/svg?seed=${username}" alt="${username}" class="dm-header-avatar">
            <div>
                <h4>${username}</h4>
                <span class="dm-header-status">Direct Message</span>
            </div>
        </div>
    `;

    // Show input area
    document.getElementById('chat-input-area').style.display = 'flex';

    // Clear messages
    document.getElementById('chat-messages').innerHTML = '';

    // Create or get the DM room
    let roomId = existingRoomId;
    if (!roomId) {
        try {
            const res = await authFetch('/api/v1/dm/start', {
                method: 'POST',
                body: JSON.stringify({
                    user1_id: currentUser.id,
                    user2_id: userId,
                    username1: currentUser.username,
                    username2: username
                })
            });
            const room = await res.json();
            roomId = room.id;
        } catch (err) {
            console.error('Failed to create DM room:', err);
            return;
        }
    }

    // Switch to this room's WebSocket
    if (currentRoomId !== roomId) {
        currentRoomId = roomId;

        // Load message history
        await loadRoomMessages(roomId);

        // Connect WebSocket to this room
        connectWebSocket();
    }

    // Focus input
    document.getElementById('chat-input').focus();
}

// ─── Load Room Messages ───
async function loadRoomMessages(roomId) {
    try {
        const res = await authFetch(`/api/v1/rooms/${roomId}/messages`);
        const messages = await res.json();

        const container = document.getElementById('chat-messages');
        container.innerHTML = '';

        if (messages && messages.length > 0) {
            messages.forEach(msg => {
                appendChatMessage(msg.username, msg.content, msg.timestamp);
            });
        } else {
            container.innerHTML = `
                <div class="chat-dm-start">
                    <span>👋</span>
                    <p>This is the beginning of your conversation. Say hello!</p>
                </div>
            `;
        }
        container.scrollTop = container.scrollHeight;
    } catch (err) {
        console.error('Failed to load messages:', err);
    }
}

// ─── WebSocket Connection (with JWT token) ───
function connectWebSocket() {
    if (!currentUser || !currentRoomId) return;

    if (ws) {
        ws.close();
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const token = getAuthToken();
    let wsUrl = `${protocol}//${window.location.host}/api/v1/ws?username=${encodeURIComponent(currentUser.username)}&room=${encodeURIComponent(currentRoomId)}`;
    
    // Append JWT token for server-side auth
    if (token) {
        wsUrl += `&token=${encodeURIComponent(token)}`;
    }

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log('✅ WebSocket connected to', currentRoomId);
        reconnectAttempts = 0;
    };

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);

            if (msg.type === 'system') {
                // Don't show system messages in DM (join/leave noise)
                return;
            } else if (msg.type === 'chat') {
                appendChatMessage(msg.username, msg.content, new Date().toISOString());
            }

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
    if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) return;

    reconnectAttempts++;
    if (reconnectTimer) clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(() => {
        if (currentUser && currentRoomId) connectWebSocket();
    }, RECONNECT_DELAY);
}

// ─── Send Message ───
function sendMessage() {
    const input = document.getElementById('chat-input');
    const content = input.value.trim();
    if (!content || !ws || ws.readyState !== WebSocket.OPEN || !currentUser) return;

    const msg = {
        type: 'chat',
        content: content,
        username: currentUser.username,
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
function appendChatMessage(username, content, timestamp) {
    const container = document.getElementById('chat-messages');
    if (!container) return;

    // Remove the "say hello" placeholder if present
    const startMsg = container.querySelector('.chat-dm-start');
    if (startMsg) startMsg.remove();

    // Remove the empty state if present
    const emptyState = container.querySelector('.chat-empty-state');
    if (emptyState) emptyState.remove();

    const time = timestamp ? new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '';
    const isMe = currentUser && username === currentUser.username;

    const msgEl = document.createElement('div');
    msgEl.className = `chat-msg ${isMe ? 'chat-msg-me' : 'chat-msg-them'}`;
    msgEl.innerHTML = `
        <div class="chat-msg-avatar" style="background:${isMe ? 'linear-gradient(135deg, #10b981, #059669)' : 'var(--gradient-primary)'}">
            ${username.slice(0, 2).toUpperCase()}
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
    container.scrollTop = container.scrollHeight;
}

function escapeHtmlChat(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
