# DevConnect — AI-Powered Developer Collaboration Hub ⚡

> **Find your tribe. Build together. Ship faster.**

DevConnect is a full-stack developer networking platform that uses AI to intelligently match engineers based on skills, interests, and project needs. Built with a Go backend, real-time WebSockets, and multiple AI engines, it combines developer discovery, project collaboration, and private messaging into one polished experience.

---

## ✨ Features

### 🔐 Authentication & Security
- **JWT-based auth** — Secure token authentication with automatic session management
- **GitHub-verified registration** — Every user must provide a **real GitHub username**, validated against the GitHub API in real-time
- **Email validation** — Strict format validation + duplicate detection at the database level
- **Protected routes** — All platform data (developers, projects, chat) hidden behind auth wall
- **Auto-logout on token expiry** — Seamless session handling

### 👨‍💻 Developer Network
- Browse **20+ developer profiles** with skills, bios, locations, and GitHub links
- **Real GitHub avatars** — Pulled automatically from `github.com/{username}.png`
- Search/filter by skill, name, or location
- Direct message any developer with one click

### 🚀 Project Collaboration
- Post open-source projects with tech stack, description, and team size
- Join projects with member tracking and capacity limits
- Search projects by name or technology
- View detailed project pages with team member cards

### 🧠 AI Matchmaking (RAG)
- **Semantic matching** using Google Gemini embeddings (`text-embedding-004`) + PostgreSQL `pgvector`
- Find the best collaborators for any developer based on skill/interest similarity
- Find ideal contributors for any project based on tech stack alignment
- Visual match scores with AI-generated match reasons

### 💬 Real-Time Direct Messaging
- Private 1-on-1 chat powered by **WebSockets** (Go channels + goroutines)
- People list with search — start a DM with any developer
- Message history persistence (FakeDB or PostgreSQL)
- Live typing and instant message delivery

### 🤖 AI Assistant (Groq-Powered)
- Platform-aware chatbot that knows all developers and projects on DevConnect
- Ask questions like *"Who works with Kubernetes?"* or *"Tell me about arjun_dev"*
- Powered by **Groq** (LLaMA) for fast inference
- Rich formatted responses with markdown support

---

## 🛠️ Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Go 1.24, Gin Framework, gorilla/websocket |
| **Auth** | JWT (JSON Web Tokens) with middleware |
| **Database** | PostgreSQL + pgvector (production) / In-memory FakeDB (development) |
| **AI — Matching** | Google Gemini API (`text-embedding-004`, `gemini-2.5-flash`) |
| **AI — Chatbot** | Groq API (LLaMA 3) |
| **Frontend** | Vanilla HTML5, CSS3 (custom glassmorphism design system), JavaScript |
| **Infra** | Docker, Docker Compose, Kubernetes |

---

## 📁 Project Structure

```
ai-match/
├── cmd/server/main.go          # Application entry point
├── internal/
│   ├── handlers/routes.go      # API route handlers (REST + WebSocket)
│   ├── middleware/jwt.go       # JWT auth middleware
│   ├── models/                 # Data models (User, Project, Message, etc.)
│   ├── repository/
│   │   ├── fake_db.go          # In-memory database for development
│   │   ├── postgres.go         # PostgreSQL + pgvector repository
│   │   └── seed.go             # Seed data (20 developers, 8 projects)
│   └── service/
│       ├── gemini.go           # Gemini AI embedding service
│       ├── groq.go             # Groq AI chat service
│       ├── match.go            # AI matching engine
│       └── hub.go              # WebSocket hub (Go channels)
├── web/
│   ├── index.html              # Single-page app (auth wall + main app)
│   ├── app.js                  # Core application logic + SPA navigation
│   ├── chat.js                 # WebSocket chat client
│   └── style.css               # Glassmorphism design system
├── k8s/                        # Kubernetes deployment manifests
├── Dockerfile                  # Multi-stage Docker build (~20MB image)
├── docker-compose.yml          # Docker Compose (app + PostgreSQL + pgvector)
└── .env                        # Environment configuration
```

---

## 🔌 API Endpoints

### Public (No Auth)
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/health` | Health check + system status |
| `GET` | `/api/v1/stats` | Platform statistics |
| `POST` | `/api/v1/auth/login` | Login (returns JWT token) |
| `POST` | `/api/v1/users` | Register (validates email + verifies GitHub via API) |

### Protected (JWT Required)
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/users` | List all developers |
| `GET` | `/api/v1/users/:id` | Get developer profile |
| `POST` | `/api/v1/projects` | Create a new project |
| `GET` | `/api/v1/projects` | List all projects |
| `GET` | `/api/v1/projects/:id` | Get project details |
| `POST` | `/api/v1/projects/:id/join` | Join a project |
| `GET` | `/api/v1/projects/:id/members` | Get project members |
| `GET` | `/api/v1/match/user/:id` | AI match — find collaborators |
| `GET` | `/api/v1/match/project/:id` | AI match — find contributors |
| `GET` | `/api/v1/connections/user/:id` | Get user connections |
| `POST` | `/api/v1/ai/chat` | AI assistant chat |
| `POST` | `/api/v1/dm/start` | Start a DM conversation |
| `GET` | `/api/v1/dm/list/:userId` | List user's DM conversations |
| `GET` | `/api/v1/ws` | WebSocket connection |

---

## 🚀 Quick Start

### Prerequisites
- [Go 1.24+](https://go.dev/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (optional, for PostgreSQL)
- A [Groq API Key](https://console.groq.com/) (for AI chatbot)
- A [Google Gemini API Key](https://aistudio.google.com/apikey) (for AI matching)

### Option 1: Run Locally (In-Memory DB)

```bash
# Clone the repo
git clone https://github.com/shivansh-gohem/ai-match.git
cd ai-match

# Configure environment
cp .env.example .env
# Edit .env and add your API keys

# Run
go mod download
go build ./... && go run cmd/server/main.go
```

Open **http://localhost:8080** — Login with `arjun_dev` / `password123` or register a new account with your real GitHub username.

### Option 2: Docker Compose (PostgreSQL + pgvector)

```bash
# Clone & configure .env (add your API keys)
git clone https://github.com/shivansh-gohem/ai-match.git
cd ai-match

# Start everything (builds Go app + spins up PostgreSQL with pgvector)
docker compose up --build -d
```

Visit **http://localhost:8080**

### Option 3: Kubernetes

```bash
# Create secrets
kubectl create secret generic devconnect-secrets \
  --from-literal=db-password='your_secure_db_password' \
  --from-literal=gemini-api-key='your_gemini_api_key' \
  --from-literal=groq-api-key='your_groq_api_key' \
  --from-literal=jwt-secret='your_jwt_secret'

# Deploy
kubectl apply -f k8s/db.yaml
kubectl apply -f k8s/app.yaml

# Access
kubectl port-forward service/devconnect-app-service 8080:8080
```

---

## ⚙️ Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `PORT` | Server port (default: `8080`) | No |
| `DB_TYPE` | `fake` (in-memory) or `postgres` | No (default: `fake`) |
| `GEMINI_API_KEY` | Google Gemini API key for embeddings & matching | Yes (for AI matching) |
| `GROQ_API_KEY` | Groq API key for AI assistant chatbot | Yes (for AI chatbot) |
| `JWT_SECRET` | Secret key for signing JWT tokens | Yes |
| `DB_USER` | PostgreSQL username | Only if `DB_TYPE=postgres` |
| `DB_PASSWORD` | PostgreSQL password | Only if `DB_TYPE=postgres` |
| `DB_HOST` | PostgreSQL host | Only if `DB_TYPE=postgres` |
| `DB_PORT` | PostgreSQL port | Only if `DB_TYPE=postgres` |
| `DB_NAME` | PostgreSQL database name | Only if `DB_TYPE=postgres` |

---

## 🔒 Registration Validation

DevConnect enforces strict registration to ensure every profile is legitimate:

1. **Email** — Regex format validation + `UNIQUE` constraint (no duplicate emails)
2. **GitHub Username** — Format validation (1-39 chars, alphanumeric + hyphens) + **live API verification** against `api.github.com/users/{id}`
3. **GitHub Avatar** — Automatically fetched from `github.com/{username}.png`
4. **Password** — Required field (stored as hash in production)

Fake GitHub usernames are **rejected** with a clear error message.

---

## 🤝 Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## 📄 License

[MIT](https://choosealicense.com/licenses/mit/)
