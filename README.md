# DevConnect: AI-Powered Developer Matchmaker ⚡

DevConnect is an intelligent platform designed to help developers find collaborators, open-source projects, and mentors. Built with an event-driven Go backend, real-time WebSockets, and Google's Gemini AI, DevConnect uses vector embeddings (RAG) to instantly match users with developers and projects based on their skills and interests.

## ✨ Features
* **🧠 AI Matchmaking (RAG):** Uses `gemini-2.5-flash` embeddings and PostgreSQL (`pgvector`) to find perfect developer and project matches based on semantic similarity of skills and bios.
* **💬 Real-Time Chat Rooms:** Live, concurrent chat rooms built with Go Channels and WebSockets.
* **🤖 AI Chatbot:** An integrated Gemini-powered AI assistant ready to answer programming and system design questions.
* **🌐 Developer & Project Discovery:** Browse open projects or explore profiles of developers worldwide.
* **🔐 Authentication:** User registration and login utilizing secure sessions and local storage.
* **🐳 Cloud-Ready:** Fully containerized with Docker, Docker Compose, and Kubernetes manifests included.

## 🛠️ Tech Stack
* **Backend:** Go (`net/http`, `gorilla/websocket`), Gin Framework
* **Database:** PostgreSQL with `pgvector`
* **AI/LLM:** Google Gemini AI API (`text-embedding-004`, `gemini-2.5-flash`)
* **Frontend:** Vanilla HTML, CSS (Custom Glassmorphism Design System), JavaScript
* **Infrastructure:** Docker, Docker Compose, Kubernetes

---

## 🚀 Quick Start (Local Development)

### Prerequisites
* [Go 1.24+](https://go.dev/)
* [Docker Desktop](https://www.docker.com/products/docker-desktop/)
* A [Google Gemini API Key](https://aistudio.google.com/apikey)

### 1. Clone & Configure
```bash
git clone https://github.com/your-username/ai-match.git
cd ai-match

# Create your environment file
cp .env.example .env
```
Open `.env` and insert your Gemini API Key:
```env
PORT=8080
DB_TYPE=postgres
GEMINI_API_KEY=your_api_key_here
```

### 2. Run with Docker Compose
The easiest way to run the entire cluster (Go Backend + Postgres Database + pgvector).
```bash
docker compose up --build -d
```
Visit `http://localhost:8080/` in your browser.

---

## ☸️ Kubernetes Deployment

If you want to host DevConnect on a Kubernetes cluster (e.g., Minikube, EKS, GKE), deployment files are included in the `k8s/` directory.

### 1. Create Secrets
```bash
kubectl create secret generic devconnect-secrets \
  --from-literal=db-password='your_secure_db_password' \
  --from-literal=gemini-api-key='your_gemini_api_key'
```

### 2. Apply Manifests
Deploy the database and application:
```bash
kubectl apply -f k8s/db.yaml
kubectl apply -f k8s/app.yaml
```

### 3. Verify
```bash
kubectl get pods
kubectl port-forward service/devconnect-app-service 8080:8080
```

---

## 🧑‍💻 Manual Build (No Docker)

If you just want to run the Go server manually:
1. Ensure your `.env` is set to `DB_TYPE=fake` (uses an in-memory test database) or spin up your own Postgres instance.
2. Install dependencies: `go mod download`
3. Run the server: `go run cmd/server/main.go`

## 🤝 Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## 📄 License
[MIT](https://choosealicense.com/licenses/mit/)
