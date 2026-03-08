/* ═══════════════════════════════════════════════
   DevConnect — Main Application Logic
   SPA navigation, API calls, dynamic rendering
   ═══════════════════════════════════════════════ */

const API_BASE = '/api/v1';

let currentUser = null;

// Initialize Auth State on load
document.addEventListener('DOMContentLoaded', () => {
    const savedUser = localStorage.getItem('currentUser');
    if (savedUser) {
        currentUser = JSON.parse(savedUser);
        updateAuthUI();
    }
});

function showAuthModal() {
    if (currentUser) {
        currentUser = null;
        localStorage.removeItem('currentUser');
        updateAuthUI();
        alert('Logged out successfully.');
        return;
    }

    document.getElementById('auth-modal').classList.add('active');
    switchAuthTab('login');
}

function switchAuthTab(tab) {
    if (tab === 'login') {
        document.getElementById('login-form').style.display = 'block';
        document.getElementById('register-form').style.display = 'none';
        document.getElementById('tab-login').classList.replace('btn-secondary', 'btn-primary');
        document.getElementById('tab-register').classList.replace('btn-primary', 'btn-secondary');
        document.getElementById('auth-title').innerText = 'Login';
    } else {
        document.getElementById('login-form').style.display = 'none';
        document.getElementById('register-form').style.display = 'block';
        document.getElementById('tab-register').classList.replace('btn-secondary', 'btn-primary');
        document.getElementById('tab-login').classList.replace('btn-primary', 'btn-secondary');
        document.getElementById('auth-title').innerText = 'Register';
    }
}

async function handleLogin(e) {
    e.preventDefault();
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;

    try {
        const res = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, password })
        });

        if (!res.ok) {
            const data = await res.json();
            throw new Error(data.error || 'Login failed');
        }

        const data = await res.json();
        currentUser = data; // Assuming data is the user object
        localStorage.setItem('currentUser', JSON.stringify(currentUser));

        closeModal('auth-modal');
        updateAuthUI();
        loadDevelopers();
        alert('Welcome back, ' + currentUser.username + '!');
    } catch (err) {
        alert(err.message);
    }
}

async function handleRegister(e) {
    e.preventDefault();
    const username = document.getElementById('reg-username').value;
    const email = document.getElementById('reg-email').value;
    const password = document.getElementById('reg-password').value;
    const skillsRaw = document.getElementById('reg-skills').value;

    const skills = skillsRaw.split(',').map(s => s.trim()).filter(s => s);

    try {
        const res = await fetch(`${API_BASE}/users`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ username, email, password, skills, interests: skills })
        });

        if (!res.ok) {
            const data = await res.json();
            throw new Error(data.error || 'Registration failed');
        }

        const data = await res.json();
        currentUser = data;
        localStorage.setItem('currentUser', JSON.stringify(currentUser));

        closeModal('auth-modal');
        updateAuthUI();
        loadDevelopers();
        alert('Registration successful! Welcome to DevConnect.');
    } catch (err) {
        alert(err.message);
    }
}

function updateAuthUI() {
    const authBtn = document.getElementById('auth-btn');
    if (currentUser) {
        authBtn.innerText = 'Logout (' + currentUser.username + ')';
        authBtn.classList.replace('btn-secondary', 'btn-primary');
    } else {
        authBtn.innerText = 'Login / Register';
        authBtn.classList.replace('btn-primary', 'btn-secondary');
    }
}

// ─── SPA Navigation ───
function showSection(sectionName) {
    // Hide all sections
    document.querySelectorAll('.section').forEach(s => s.classList.remove('active'));

    // Show target section
    const target = document.getElementById(`section-${sectionName}`);
    if (target) {
        target.classList.add('active');
    }

    // Update nav links
    document.querySelectorAll('.nav-link').forEach(l => l.classList.remove('active'));
    const navLink = document.querySelector(`[data-section="${sectionName}"]`);
    if (navLink) navLink.classList.add('active');

    // Load data for the section
    switch (sectionName) {
        case 'developers': loadDevelopers(); break;
        case 'projects': loadProjects(); break;
        case 'matchmaker': loadMatchUsers(); break;
        case 'chat': loadRooms(); break;
    }
}

// Setup nav link clicks
document.querySelectorAll('.nav-link').forEach(link => {
    link.addEventListener('click', (e) => {
        e.preventDefault();
        showSection(link.dataset.section);
    });
});

// ─── Load Stats ───
async function loadStats() {
    try {
        const res = await fetch(`${API_BASE}/stats`);
        const data = await res.json();

        animateNumber('stat-devs', data.total_developers);
        animateNumber('stat-projects', data.total_projects);
        animateNumber('stat-rooms', data.total_rooms);
        document.getElementById('online-num').textContent = data.online_now;
    } catch (err) {
        console.error('Failed to load stats:', err);
    }
}

function animateNumber(elementId, target) {
    const el = document.getElementById(elementId);
    if (!el) return;

    let current = 0;
    const increment = Math.ceil(target / 30);
    const timer = setInterval(() => {
        current += increment;
        if (current >= target) {
            current = target;
            clearInterval(timer);
        }
        el.textContent = current;
    }, 40);
}

// ─── Developers ───
let allDevelopers = [];

async function loadDevelopers() {
    try {
        const res = await fetch(`${API_BASE}/users`);
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
                ${dev.github_url ? `<a href="${dev.github_url}" target="_blank" class="btn btn-secondary btn-small">GitHub ↗</a>` : ''}
            </div>
        </div>
    `).join('');
}

// Developer search
const devSearch = document.getElementById('dev-search');
if (devSearch) {
    devSearch.addEventListener('input', (e) => {
        const q = e.target.value.toLowerCase();
        const filtered = allDevelopers.filter(d =>
            d.username.toLowerCase().includes(q) ||
            d.bio.toLowerCase().includes(q) ||
            d.location.toLowerCase().includes(q) ||
            d.skills.some(s => s.toLowerCase().includes(q))
        );
        renderDevelopers(filtered);
    });
}

// ─── Projects ───
async function loadProjects() {
    try {
        const res = await fetch(`${API_BASE}/projects`);
        const projects = await res.json();
        renderProjects(projects);
    } catch (err) {
        console.error('Failed to load projects:', err);
    }
}

function renderProjects(projects) {
    const grid = document.getElementById('projects-grid');
    if (!grid) return;

    grid.innerHTML = projects.map(p => `
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
                <span>👥 Max ${p.max_members} members</span>
            </div>
            <button class="btn btn-primary btn-small" onclick="findProjectMatches('${p.id}')" style="margin-top:0.5rem;">
                🧠 Find Contributors
            </button>
        </div>
    `).join('');
}

function showCreateProject() {
    document.getElementById('create-project-modal').classList.add('active');
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('active');
}

async function createProject(e) {
    e.preventDefault();

    const techStr = document.getElementById('project-tech').value;
    const techStack = techStr.split(',').map(s => s.trim()).filter(s => s);

    const body = {
        title: document.getElementById('project-title').value,
        description: document.getElementById('project-desc').value,
        tech_stack: techStack,
        owner_id: 'user001', // Default for MVP demo
        max_members: parseInt(document.getElementById('project-members').value) || 5,
    };

    try {
        const res = await fetch(`${API_BASE}/projects`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });

        if (res.ok) {
            closeModal('create-project-modal');
            document.getElementById('create-project-form').reset();
            loadProjects();
        }
    } catch (err) {
        console.error('Failed to create project:', err);
    }
}

// ─── AI Matchmaker ───
async function loadMatchUsers() {
    try {
        const res = await fetch(`${API_BASE}/users`);
        const users = await res.json();
        renderMatchUserList(users);
    } catch (err) {
        console.error('Failed to load match users:', err);
    }
}

function renderMatchUserList(users) {
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
}

function selectMatchUser(userId, element) {
    document.querySelectorAll('.match-user-item').forEach(i => i.classList.remove('selected'));
    element.classList.add('selected');
    findMatchesFor(userId);
}

async function findMatchesFor(userId) {
    // Switch to matchmaker section if not there
    const matchSection = document.getElementById('section-matchmaker');
    if (!matchSection.classList.contains('active')) {
        showSection('matchmaker');
        // Wait for render then select
        setTimeout(() => {
            const item = document.getElementById(`match-item-${userId}`);
            if (item) {
                document.querySelectorAll('.match-user-item').forEach(i => i.classList.remove('selected'));
                item.classList.add('selected');
            }
        }, 200);
    }

    // Show loading
    document.getElementById('match-placeholder').style.display = 'none';
    document.getElementById('match-results').style.display = 'none';
    document.getElementById('match-loading').style.display = 'block';

    try {
        const res = await fetch(`${API_BASE}/match/user/${userId}`);
        const data = await res.json();
        renderMatchResults(data.matches || []);
    } catch (err) {
        console.error('Failed to find matches:', err);
        document.getElementById('match-loading').style.display = 'none';
        document.getElementById('match-placeholder').style.display = 'flex';
    }
}

async function findProjectMatches(projectId) {
    showSection('matchmaker');

    document.getElementById('match-placeholder').style.display = 'none';
    document.getElementById('match-results').style.display = 'none';
    document.getElementById('match-loading').style.display = 'block';

    try {
        const res = await fetch(`${API_BASE}/match/project/${projectId}`);
        const data = await res.json();
        renderMatchResults(data.matches || []);
    } catch (err) {
        console.error('Failed to find project matches:', err);
    }
}

function renderMatchResults(matches) {
    document.getElementById('match-loading').style.display = 'none';
    document.getElementById('match-results').style.display = 'block';

    const container = document.getElementById('match-cards');
    if (matches.length === 0) {
        container.innerHTML = '<p style="color:var(--text-muted);text-align:center;padding:2rem;">No matches found for this profile.</p>';
        return;
    }

    container.innerHTML = matches.map(m => `
        <div class="match-card">
            <img src="${m.user.avatar_url}" alt="${m.user.username}">
            <div class="match-card-info">
                <h4>
                    ${m.user.username}
                    <span class="match-score">${Math.round(m.score * 100)}% match</span>
                </h4>
                <p class="match-reason">${m.reason}</p>
                <div class="dev-skills" style="margin-top:0.5rem;">
                    ${m.user.skills.slice(0, 4).map(s => `<span class="skill-tag">${s}</span>`).join('')}
                </div>
            </div>
        </div>
    `).join('');
}

// ─── AI Assistant ───
async function sendAIMessage() {
    const input = document.getElementById('ai-input');
    const message = input.value.trim();
    if (!message) return;

    const messagesDiv = document.getElementById('ai-messages');

    // Add user message
    messagesDiv.innerHTML += `
        <div class="ai-message ai-user" style="opacity:1; animation: slideUpFade 0.4s forwards;">
            <div class="ai-avatar">👤</div>
            <div class="ai-bubble">${escapeHtml(message)}</div>
        </div>
    `;

    input.value = '';

    // Add typing indicator
    messagesDiv.innerHTML += `
        <div class="ai-message ai-bot" id="ai-typing" style="opacity:1; animation: slideUpFade 0.4s forwards;">
            <div class="ai-avatar">⚡</div>
            <div class="ai-bubble ai-typing">
                <span></span><span></span><span></span>
            </div>
        </div>
    `;
    messagesDiv.scrollTop = messagesDiv.scrollHeight;

    try {
        const res = await fetch(`${API_BASE}/ai/chat`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message }),
        });
        const data = await res.json();

        if (!res.ok) {
            throw new Error(data.error || 'Failed to get response');
        }

        // Remove typing indicator
        const typingEl = document.getElementById('ai-typing');
        if (typingEl) typingEl.remove();

        // Add AI response
        messagesDiv.innerHTML += `
            <div class="ai-message ai-bot">
                <div class="ai-avatar">🤖</div>
                <div class="ai-bubble">${formatAIResponse(data.message)}</div>
            </div>
        `;
    } catch (err) {
        const typingEl = document.getElementById('ai-typing');
        if (typingEl) typingEl.remove();

        messagesDiv.innerHTML += `
            <div class="ai-message ai-bot">
                <div class="ai-avatar">🤖</div>
                <div class="ai-bubble">Sorry, I had trouble processing that. Please try again!</div>
            </div>
        `;
    }

    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

function handleAIKeypress(e) {
    if (e.key === 'Enter') sendAIMessage();
}

function formatAIResponse(text) {
    // Basic markdown-like formatting
    return escapeHtml(text)
        .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
        .replace(/\*(.*?)\*/g, '<em>$1</em>')
        .replace(/`(.*?)`/g, '<code style="background:rgba(99,102,241,0.15);padding:0.15rem 0.4rem;border-radius:4px;font-size:0.85em;">$1</code>')
        .replace(/\n/g, '<br>');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ─── Init ───
document.addEventListener('DOMContentLoaded', () => {
    loadStats();

    // Refresh stats every 30 seconds
    setInterval(loadStats, 30000);
});
