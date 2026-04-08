/* ═══════════════════════════════════════════════
   DevConnect — Main Application Logic
   TWO VIEWS: Auth Wall (logged out) vs Main App (logged in)
   JWT-based authentication
   ═══════════════════════════════════════════════ */

const API_BASE = '/api/v1';

let currentUser = null;
let authToken = null;
let allDevelopers = [];
let allProjects = [];

// ─── Boot ───
document.addEventListener('DOMContentLoaded', () => {
    const savedUser = localStorage.getItem('currentUser');
    const savedToken = localStorage.getItem('authToken');
    if (savedUser && savedToken) {
        try {
            currentUser = JSON.parse(savedUser);
            authToken = savedToken;
        } catch (e) {
            localStorage.removeItem('currentUser');
            localStorage.removeItem('authToken');
            currentUser = null;
            authToken = null;
        }
    }
    applyAuthState();
});

// ═══════════════════════════════════════
// AUTHENTICATED FETCH — sends JWT token
// ═══════════════════════════════════════

async function authFetch(url, options = {}) {
    if (!options.headers) {
        options.headers = {};
    }
    // Inject Bearer token
    if (authToken) {
        options.headers['Authorization'] = `Bearer ${authToken}`;
    }
    // Default content-type for POST/PUT
    if (!options.headers['Content-Type'] && options.body) {
        options.headers['Content-Type'] = 'application/json';
    }

    const res = await fetch(url, options);

    // Auto-logout on 401
    if (res.status === 401) {
        console.warn('401 Unauthorized — logging out');
        handleLogout();
        showNotification('Session expired. Please login again.', 'error');
        throw new Error('Unauthorized');
    }

    return res;
}

// ═══════════════════════════════════════
// AUTH STATE — Controls entire page view
// ═══════════════════════════════════════

function applyAuthState() {
    const authWall = document.getElementById('auth-wall');
    const mainApp = document.getElementById('main-app');

    if (currentUser && authToken) {
        // LOGGED IN: hide auth wall, show main app
        authWall.style.display = 'none';
        mainApp.style.display = 'block';
        document.getElementById('auth-btn').textContent = 'Logout (' + currentUser.username + ')';
        // Load initial data
        loadDevelopers();
        loadStats();
        setInterval(loadStats, 30000);
    } else {
        // NOT LOGGED IN: show auth wall, hide EVERYTHING else
        authWall.style.display = 'flex';
        mainApp.style.display = 'none';
    }
}

function handleLogout() {
    currentUser = null;
    authToken = null;
    localStorage.removeItem('currentUser');
    localStorage.removeItem('authToken');
    // Close any WebSocket
    if (typeof ws !== 'undefined' && ws) {
        ws.close();
        ws = null;
    }
    applyAuthState();
    showNotification('Logged out. See you! 👋');
}

// ═══════════════════════════════════════
// AUTH WALL — Login / Register
// ═══════════════════════════════════════

function switchWallTab(tab) {
    const loginForm = document.getElementById('wall-login-form');
    const registerForm = document.getElementById('wall-register-form');
    const tabLogin = document.getElementById('wall-tab-login');
    const tabRegister = document.getElementById('wall-tab-register');

    if (tab === 'login') {
        loginForm.style.display = 'block';
        registerForm.style.display = 'none';
        tabLogin.classList.add('active');
        tabRegister.classList.remove('active');
    } else {
        loginForm.style.display = 'none';
        registerForm.style.display = 'block';
        tabRegister.classList.add('active');
        tabLogin.classList.remove('active');
    }
}

async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value.trim();
    const password = document.getElementById('login-password').value.trim();

    if (!username || !password) {
        showNotification('Please enter username and password', 'error');
        return;
    }

    try {
        const res = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        const data = await res.json();

        if (!res.ok) {
            throw new Error(data.error || 'Login failed');
        }

        // New JWT response: { user: {...}, token: "..." }
        currentUser = data.user;
        authToken = data.token;
        localStorage.setItem('currentUser', JSON.stringify(currentUser));
        localStorage.setItem('authToken', authToken);
        applyAuthState();
        showNotification('Welcome back, ' + currentUser.username + '! 🚀');
    } catch (err) {
        showNotification(err.message, 'error');
    }
}

async function handleRegister(e) {
    e.preventDefault();
    const username = document.getElementById('reg-username').value.trim();
    const email = document.getElementById('reg-email').value.trim();
    const password = document.getElementById('reg-password').value.trim();
    const githubId = document.getElementById('reg-github').value.trim();
    const skillsRaw = document.getElementById('reg-skills').value;
    const skills = skillsRaw.split(',').map(s => s.trim()).filter(s => s);

    const emailError = document.getElementById('email-error');
    const githubError = document.getElementById('github-error');
    const submitBtn = document.querySelector('#wall-register-form button[type="submit"]');
    let hasError = false;

    // Frontend email validation
    const emailRegex = /^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$/;
    if (!emailRegex.test(email)) {
        emailError.textContent = 'Please enter a valid email address';
        emailError.style.display = 'block';
        hasError = true;
    } else {
        emailError.style.display = 'none';
    }

    // Frontend GitHub ID validation (format only — backend does real API check)
    const githubRegex = /^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?$/;
    if (!githubId || githubId.length > 39 || !githubRegex.test(githubId)) {
        githubError.textContent = 'Enter a valid GitHub username (1-39 chars, letters, numbers, hyphens)';
        githubError.style.display = 'block';
        hasError = true;
    } else {
        githubError.style.display = 'none';
    }

    if (hasError) return;

    if (!username || !password || skills.length === 0) {
        showNotification('Please fill in all fields', 'error');
        return;
    }

    // Show verifying state
    submitBtn.disabled = true;
    submitBtn.textContent = '🔍 Verifying GitHub...';

    try {
        // Register creates the user (public endpoint, no token needed)
        // Backend will verify GitHub username exists via GitHub API
        const res = await fetch(`${API_BASE}/users`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, email, password, skills, interests: skills, github_id: githubId })
        });

        const data = await res.json();

        if (!res.ok) {
            // Show email errors inline
            if (data.error && data.error.toLowerCase().includes('email')) {
                emailError.textContent = data.error;
                emailError.style.display = 'block';
                submitBtn.disabled = false;
                submitBtn.textContent = 'Create Account →';
                return;
            }
            // Show GitHub errors inline
            if (data.error && (data.error.toLowerCase().includes('github') || data.error.toLowerCase().includes('does not exist'))) {
                githubError.textContent = data.error;
                githubError.style.display = 'block';
                submitBtn.disabled = false;
                submitBtn.textContent = 'Create Account →';
                return;
            }
            submitBtn.disabled = false;
            submitBtn.textContent = 'Create Account →';
            throw new Error(data.error || 'Registration failed');
        }

        submitBtn.textContent = '✅ Verified! Logging in...';

        // After registration, login to get a JWT
        const loginRes = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        const loginData = await loginRes.json();

        if (!loginRes.ok) {
            throw new Error(loginData.error || 'Auto-login after registration failed');
        }

        currentUser = loginData.user;
        authToken = loginData.token;
        localStorage.setItem('currentUser', JSON.stringify(currentUser));
        localStorage.setItem('authToken', authToken);
        applyAuthState();
        showNotification('Welcome to DevConnect, ' + currentUser.username + '! 🎉');
    } catch (err) {
        submitBtn.disabled = false;
        submitBtn.textContent = 'Create Account →';
        showNotification(err.message, 'error');
    }
}

// ═══════════════════════════════════════
// TOAST NOTIFICATION
// ═══════════════════════════════════════

function showNotification(message, type = 'success') {
    const existing = document.querySelector('.toast-notification');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.className = `toast-notification toast-${type}`;
    toast.textContent = message;
    document.body.appendChild(toast);

    requestAnimationFrame(() => toast.classList.add('show'));
    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => toast.remove(), 400);
    }, 3000);
}

// ═══════════════════════════════════════
// SPA NAVIGATION (only works when logged in)
// ═══════════════════════════════════════

function showSection(sectionName) {
    if (!currentUser) return; // safety guard

    document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));
    const target = document.getElementById(`section-${sectionName}`);
    if (target) target.classList.add('active');

    document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));
    const navLink = document.querySelector(`[data-section="${sectionName}"]`);
    if (navLink) navLink.classList.add('active');

    switch (sectionName) {
        case 'developers': loadDevelopers(); break;
        case 'projects': loadProjects(); break;
        case 'matchmaker': loadMatchUsers(); break;
        case 'chat': loadPeopleForDM(); break;
    }
}

// Nav click handlers
document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('.nav-link').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            if (!currentUser) return;
            showSection(link.dataset.section);
        });
    });
});

// ═══════════════════════════════════════
// STATS
// ═══════════════════════════════════════

async function loadStats() {
    try {
        // Stats is public — no auth needed
        const res = await fetch(`${API_BASE}/stats`);
        const data = await res.json();
        document.getElementById('online-num').textContent = data.online_now || 0;
    } catch (err) {
        console.error('Stats error:', err);
    }
}

// ═══════════════════════════════════════
// DEVELOPERS (protected)
// ═══════════════════════════════════════

async function loadDevelopers() {
    if (!currentUser) return;
    try {
        const res = await authFetch(`${API_BASE}/users`);
        allDevelopers = await res.json();
        renderDevelopers(allDevelopers);
    } catch (err) {
        console.error('Failed to load developers:', err);
    }
}

function renderDevelopers(developers) {
    const grid = document.getElementById('developers-grid');
    if (!grid) return;

    grid.innerHTML = developers.map(dev => `
        <div class="dev-card" id="dev-${dev.id}">
            <div class="dev-card-header">
                <img src="${dev.avatar_url}" alt="${dev.username}" class="dev-avatar" loading="lazy">
                <div class="dev-info">
                    <h3>${dev.username}</h3>
                    <span class="location">📍 ${dev.location || 'Remote'}</span>
                </div>
            </div>
            <p class="dev-bio">${dev.bio}</p>
            <div class="dev-skills">
                ${dev.skills.map(s => `<span class="skill-tag">${s}</span>`).join('')}
            </div>
            <div class="dev-actions">
                <button class="btn btn-primary btn-small" onclick="findMatchesFor('${dev.id}')">🧠 Find Matches</button>
                <button class="btn btn-secondary btn-small" onclick="openDMWith('${dev.id}', '${dev.username}')">💬 Message</button>
                ${dev.github_url ? `<a href="${dev.github_url}" target="_blank" class="btn btn-secondary btn-small">GitHub ↗</a>` : ''}
            </div>
        </div>
    `).join('');
}

// Developer search
document.addEventListener('DOMContentLoaded', () => {
    const devSearch = document.getElementById('dev-search');
    if (devSearch) {
        devSearch.addEventListener('input', (e) => {
            const q = e.target.value.toLowerCase();
            const filtered = allDevelopers.filter(d =>
                d.username.toLowerCase().includes(q) ||
                d.bio.toLowerCase().includes(q) ||
                (d.location && d.location.toLowerCase().includes(q)) ||
                d.skills.some(s => s.toLowerCase().includes(q))
            );
            renderDevelopers(filtered);
        });
    }
});

// ═══════════════════════════════════════
// PROJECTS (protected)
// ═══════════════════════════════════════

async function loadProjects() {
    if (!currentUser) return;
    try {
        const res = await authFetch(`${API_BASE}/projects`);
        allProjects = await res.json();
        renderProjects(allProjects);
    } catch (err) {
        console.error('Failed to load projects:', err);
    }
}

function renderProjects(projects) {
    const grid = document.getElementById('projects-grid');
    if (!grid) return;

    grid.innerHTML = projects.map(p => {
        const memberCount = (p.members || []).length;
        const memberNames = p.member_names || [];
        const memberAvatars = (p.members || []).slice(0, 3).map((uid, i) => {
            const name = memberNames[i] || 'dev';
            return `<img src="https://api.dicebear.com/7.x/avataaars/svg?seed=${name}" alt="${name}" class="member-avatar-small" title="${name}">`;
        }).join('');
        const isMember = currentUser && (p.members || []).includes(currentUser.id);
        const isFull = memberCount >= p.max_members;

        return `
        <div class="project-card" id="project-${p.id}">
            <span class="project-status ${p.status === 'open' ? 'status-open' : 'status-in-progress'}">
                ${p.status === 'open' ? '🟢 Open' : '🟡 In Progress'}
            </span>
            <h3>${p.title}</h3>
            <p class="project-desc">${p.description}</p>
            <div class="dev-skills">
                ${p.tech_stack.map(t => `<span class="skill-tag">${t}</span>`).join('')}
            </div>
            <div class="project-meta">
                <span>by <strong>${p.owner_name}</strong></span>
                <span class="member-count-badge">👥 ${memberCount}/${p.max_members}</span>
            </div>
            <div class="project-members-row">
                <div class="member-avatar-stack">${memberAvatars}</div>
                ${memberCount > 3 ? `<span class="member-more">+${memberCount - 3} more</span>` : ''}
            </div>
            <div class="project-actions">
                <button class="btn btn-secondary btn-small" onclick="showProjectDetail('${p.id}')">📋 Details</button>
                ${isMember
                    ? `<span class="joined-badge">✅ Joined</span>`
                    : isFull
                        ? `<span class="full-badge">🔒 Full</span>`
                        : `<button class="btn btn-primary btn-small" onclick="joinProject('${p.id}')">🤝 Join</button>`
                }
                <button class="btn btn-secondary btn-small" onclick="findProjectMatches('${p.id}')">🧠 Contributors</button>
            </div>
        </div>`;
    }).join('');
}

// Project search
document.addEventListener('DOMContentLoaded', () => {
    const projectSearch = document.getElementById('project-search');
    if (projectSearch) {
        projectSearch.addEventListener('input', (e) => {
            const q = e.target.value.toLowerCase();
            const filtered = allProjects.filter(p =>
                p.title.toLowerCase().includes(q) ||
                p.description.toLowerCase().includes(q) ||
                p.owner_name.toLowerCase().includes(q) ||
                p.tech_stack.some(t => t.toLowerCase().includes(q))
            );
            renderProjects(filtered);
        });
    }
});

async function joinProject(projectId) {
    if (!currentUser) return;
    try {
        const res = await authFetch(`${API_BASE}/projects/${projectId}/join`, {
            method: 'POST',
            body: JSON.stringify({ user_id: currentUser.id })
        });
        const data = await res.json();
        if (res.ok) {
            showNotification('Joined project! 🎉');
            loadProjects();
        } else {
            showNotification(data.error || 'Failed to join', 'error');
        }
    } catch (err) {
        showNotification('Failed to join project', 'error');
    }
}

async function showProjectDetail(projectId) {
    try {
        const [projRes, membersRes] = await Promise.all([
            authFetch(`${API_BASE}/projects/${projectId}`),
            authFetch(`${API_BASE}/projects/${projectId}/members`)
        ]);
        const project = await projRes.json();
        const members = await membersRes.json();

        const memberCards = (members || []).map(m => `
            <div class="detail-member-card">
                <img src="${m.avatar_url}" alt="${m.username}" class="detail-member-avatar">
                <div>
                    <strong>${m.username}</strong>
                    <span class="detail-member-location">📍 ${m.location || 'Remote'}</span>
                    <div class="dev-skills" style="margin-top:0.25rem;">
                        ${m.skills.slice(0, 3).map(s => `<span class="skill-tag">${s}</span>`).join('')}
                    </div>
                </div>
            </div>
        `).join('');

        const isMember = currentUser && (project.members || []).includes(currentUser.id);
        const isFull = (project.members || []).length >= project.max_members;

        document.getElementById('project-detail-title').textContent = project.title;
        document.getElementById('project-detail-body').innerHTML = `
            <span class="project-status ${project.status === 'open' ? 'status-open' : 'status-in-progress'}">
                ${project.status === 'open' ? '🟢 Open' : '🟡 In Progress'}
            </span>
            <p style="margin:1rem 0;color:var(--text-secondary);line-height:1.7;">${project.description}</p>
            <h4 style="margin-bottom:0.5rem;">🛠️ Tech Stack</h4>
            <div class="dev-skills" style="margin-bottom:1.5rem;">
                ${project.tech_stack.map(t => `<span class="skill-tag">${t}</span>`).join('')}
            </div>
            <h4 style="margin-bottom:0.75rem;">👥 Team (${(project.members||[]).length}/${project.max_members})</h4>
            <div class="detail-members-list">
                ${memberCards || '<p style="color:var(--text-muted);">No members yet.</p>'}
            </div>
            <div style="margin-top:1.5rem;text-align:center;">
                ${isMember ? `<span class="joined-badge" style="font-size:1rem;">✅ You're a member</span>`
                    : isFull ? `<span class="full-badge" style="font-size:1rem;">🔒 Full</span>`
                    : `<button class="btn btn-primary" onclick="joinProject('${project.id}');closeModal('project-detail-modal');">🤝 Join</button>`}
            </div>
        `;
        document.getElementById('project-detail-modal').classList.add('active');
    } catch (err) {
        console.error('Project detail error:', err);
    }
}

function showCreateProject() {
    document.getElementById('create-project-modal').classList.add('active');
}

function closeModal(id) {
    document.getElementById(id).classList.remove('active');
}

async function createProject(e) {
    e.preventDefault();
    if (!currentUser) return;
    const techStack = document.getElementById('project-tech').value.split(',').map(s => s.trim()).filter(s => s);
    const body = {
        title: document.getElementById('project-title').value,
        description: document.getElementById('project-desc').value,
        tech_stack: techStack,
        owner_id: currentUser.id,
        max_members: parseInt(document.getElementById('project-members').value) || 5,
    };
    try {
        const res = await authFetch(`${API_BASE}/projects`, {
            method: 'POST',
            body: JSON.stringify(body),
        });
        if (res.ok) {
            closeModal('create-project-modal');
            document.getElementById('create-project-form').reset();
            loadProjects();
            showNotification('Project posted! 🚀');
        }
    } catch (err) {
        console.error('Create project error:', err);
    }
}

// ═══════════════════════════════════════
// AI MATCHMAKER (protected)
// ═══════════════════════════════════════

async function loadMatchUsers() {
    if (!currentUser) return;
    try {
        const res = await authFetch(`${API_BASE}/users`);
        const users = await res.json();
        const list = document.getElementById('match-user-list');
        if (!list) return;
        list.innerHTML = users.map(u => `
            <div class="match-user-item" onclick="selectMatchUser('${u.id}', this)" id="match-item-${u.id}">
                <img src="${u.avatar_url}" alt="${u.username}">
                <div class="match-user-info">
                    <strong>${u.username}</strong>
                    <span>${u.skills.slice(0, 3).join(', ')}</span>
                </div>
            </div>
        `).join('');
    } catch (err) {
        console.error('Match users error:', err);
    }
}

function selectMatchUser(userId, el) {
    document.querySelectorAll('.match-user-item').forEach(i => i.classList.remove('selected'));
    el.classList.add('selected');
    findMatchesFor(userId);
}

async function findMatchesFor(userId) {
    if (!currentUser) return;
    const section = document.getElementById('section-matchmaker');
    if (!section.classList.contains('active')) {
        showSection('matchmaker');
        setTimeout(() => {
            const item = document.getElementById(`match-item-${userId}`);
            if (item) { document.querySelectorAll('.match-user-item').forEach(i => i.classList.remove('selected')); item.classList.add('selected'); }
        }, 200);
    }
    document.getElementById('match-placeholder').style.display = 'none';
    document.getElementById('match-results').style.display = 'none';
    document.getElementById('match-loading').style.display = 'block';
    try {
        const res = await authFetch(`${API_BASE}/match/user/${userId}`);
        const data = await res.json();
        renderMatchResults(data.matches || []);
    } catch (err) {
        document.getElementById('match-loading').style.display = 'none';
        document.getElementById('match-placeholder').style.display = 'flex';
    }
}

async function findProjectMatches(projectId) {
    if (!currentUser) return;
    showSection('matchmaker');
    document.getElementById('match-placeholder').style.display = 'none';
    document.getElementById('match-results').style.display = 'none';
    document.getElementById('match-loading').style.display = 'block';
    try {
        const res = await authFetch(`${API_BASE}/match/project/${projectId}`);
        const data = await res.json();
        renderMatchResults(data.matches || []);
    } catch (err) {
        console.error('Project match error:', err);
    }
}

function renderMatchResults(matches) {
    document.getElementById('match-loading').style.display = 'none';
    document.getElementById('match-results').style.display = 'block';
    const container = document.getElementById('match-cards');
    if (!matches.length) {
        container.innerHTML = '<p style="color:var(--text-muted);text-align:center;padding:2rem;">No matches found.</p>';
        return;
    }
    container.innerHTML = matches.map(m => `
        <div class="match-card">
            <img src="${m.user.avatar_url}" alt="${m.user.username}">
            <div class="match-card-info">
                <h4>${m.user.username} <span class="match-score">${Math.round(m.score * 100)}%</span></h4>
                <p class="match-reason">${m.reason}</p>
                <div class="dev-skills" style="margin-top:0.5rem;">
                    ${m.user.skills.slice(0, 4).map(s => `<span class="skill-tag">${s}</span>`).join('')}
                </div>
            </div>
        </div>
    `).join('');
}

// ═══════════════════════════════════════
// AI ASSISTANT (protected, Groq)
// ═══════════════════════════════════════

async function sendAIMessage() {
    if (!currentUser) return;
    const input = document.getElementById('ai-input');
    const message = input.value.trim();
    if (!message) return;

    const messagesDiv = document.getElementById('ai-messages');
    messagesDiv.innerHTML += `<div class="ai-message ai-user"><div class="ai-avatar">👤</div><div class="ai-bubble">${escapeHtml(message)}</div></div>`;
    input.value = '';
    messagesDiv.innerHTML += `<div class="ai-message ai-bot" id="ai-typing"><div class="ai-avatar">⚡</div><div class="ai-bubble ai-typing"><span></span><span></span><span></span></div></div>`;
    messagesDiv.scrollTop = messagesDiv.scrollHeight;

    try {
        const res = await authFetch(`${API_BASE}/ai/chat`, {
            method: 'POST',
            body: JSON.stringify({ message }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'AI error');

        const typing = document.getElementById('ai-typing');
        if (typing) typing.remove();
        messagesDiv.innerHTML += `<div class="ai-message ai-bot"><div class="ai-avatar">⚡</div><div class="ai-bubble">${formatAIResponse(data.message)}</div></div>`;
    } catch (err) {
        const typing = document.getElementById('ai-typing');
        if (typing) typing.remove();
        messagesDiv.innerHTML += `<div class="ai-message ai-bot"><div class="ai-avatar">⚡</div><div class="ai-bubble">Sorry, something went wrong. Please try again!</div></div>`;
    }
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

function handleAIKeypress(e) { if (e.key === 'Enter') sendAIMessage(); }

function formatAIResponse(text) {
    return escapeHtml(text)
        .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
        .replace(/\*(.*?)\*/g, '<em>$1</em>')
        .replace(/`(.*?)`/g, '<code style="background:rgba(99,102,241,0.15);padding:0.15rem 0.4rem;border-radius:4px;">$1</code>')
        .replace(/\n/g, '<br>');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ═══════════════════════════════════════
// DM (from developer card) — protected
// ═══════════════════════════════════════

async function openDMWith(targetUserId, targetUsername) {
    if (!currentUser) return;
    try {
        const res = await authFetch(`${API_BASE}/dm/start`, {
            method: 'POST',
            body: JSON.stringify({
                user1_id: currentUser.id,
                user2_id: targetUserId,
                username1: currentUser.username,
                username2: targetUsername
            })
        });
        const room = await res.json();
        showSection('chat');
        setTimeout(() => selectDMPerson(targetUserId, targetUsername, room.id), 300);
    } catch (err) {
        showNotification('Failed to start DM', 'error');
    }
}

// ═══════════════════════════════════════
// Helper: get auth token (used by chat.js)
// ═══════════════════════════════════════

function getAuthToken() {
    return authToken;
}
