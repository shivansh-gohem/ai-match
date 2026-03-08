package repository

import (
	"time"
	"math/rand"

	"github.com/shiva/ai-match/internal/models"
)

// Seed populates the database with fake developer profiles and projects.
func (db *FakeDB) Seed() {
	// ---------- FAKE DEVELOPERS ----------
	fakeUsers := []models.User{
		{ID: "user001", Username: "arjun_dev", Email: "arjun@devhub.io", Bio: "Full-stack engineer passionate about distributed systems and Kubernetes.", Skills: []string{"Go", "Kubernetes", "Docker", "gRPC", "PostgreSQL"}, Interests: []string{"Cloud Native", "Open Source", "System Design"}, GithubURL: "https://github.com/arjundev", Location: "Bangalore, India", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=arjun_dev"},
		{ID: "user002", Username: "sara_codes", Email: "sara@techmail.com", Bio: "Backend developer building high-performance APIs. Rust enthusiast.", Skills: []string{"Rust", "Go", "Python", "Redis", "PostgreSQL"}, Interests: []string{"Performance Engineering", "Compilers", "WebAssembly"}, GithubURL: "https://github.com/saracodes", Location: "Berlin, Germany", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=sara_codes"},
		{ID: "user003", Username: "mike_cloud", Email: "mike@cloudops.dev", Bio: "DevOps engineer automating everything. Terraform and Ansible all day.", Skills: []string{"Terraform", "AWS", "Docker", "Kubernetes", "Ansible", "Linux"}, Interests: []string{"Infrastructure as Code", "GitOps", "SRE"}, GithubURL: "https://github.com/mikecloud", Location: "San Francisco, USA", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=mike_cloud"},
		{ID: "user004", Username: "priya_ml", Email: "priya@ailab.io", Bio: "Machine learning engineer working on NLP and recommendation systems.", Skills: []string{"Python", "TensorFlow", "PyTorch", "FastAPI", "PostgreSQL"}, Interests: []string{"NLP", "RAG Systems", "LLMs", "Computer Vision"}, GithubURL: "https://github.com/priyaml", Location: "Mumbai, India", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=priya_ml"},
		{ID: "user005", Username: "leo_frontend", Email: "leo@webcraft.co", Bio: "Frontend architect building accessible and performant web apps.", Skills: []string{"React", "TypeScript", "Next.js", "Tailwind", "GraphQL"}, Interests: []string{"Web Performance", "Accessibility", "Design Systems"}, GithubURL: "https://github.com/leofrontend", Location: "London, UK", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=leo_frontend"},
		{ID: "user006", Username: "yuki_sec", Email: "yuki@secops.jp", Bio: "Security researcher and penetration tester. Bug bounty hunter.", Skills: []string{"Python", "Go", "Linux", "Wireshark", "Burp Suite"}, Interests: []string{"Cybersecurity", "Ethical Hacking", "Zero Trust"}, GithubURL: "https://github.com/yukisec", Location: "Tokyo, Japan", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=yuki_sec"},
		{ID: "user007", Username: "carlos_data", Email: "carlos@datapipe.io", Bio: "Data engineer building real-time pipelines with Apache Kafka and Spark.", Skills: []string{"Python", "Apache Kafka", "Spark", "Airflow", "SQL", "Go"}, Interests: []string{"Data Engineering", "Stream Processing", "Data Lakes"}, GithubURL: "https://github.com/carlosdata", Location: "São Paulo, Brazil", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=carlos_data"},
		{ID: "user008", Username: "nina_mobile", Email: "nina@appdev.io", Bio: "Mobile developer building cross-platform apps with Flutter and Kotlin.", Skills: []string{"Flutter", "Dart", "Kotlin", "Swift", "Firebase"}, Interests: []string{"Mobile UX", "Cross-Platform", "AR/VR"}, GithubURL: "https://github.com/ninamobile", Location: "Toronto, Canada", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=nina_mobile"},
		{ID: "user009", Username: "omar_blockchain", Email: "omar@web3labs.io", Bio: "Web3 developer building decentralized applications on Ethereum.", Skills: []string{"Solidity", "Rust", "TypeScript", "Hardhat", "Go"}, Interests: []string{"DeFi", "Smart Contracts", "Decentralized Identity"}, GithubURL: "https://github.com/omarblockchain", Location: "Dubai, UAE", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=omar_blockchain"},
		{ID: "user010", Username: "emma_devrel", Email: "emma@techtalks.dev", Bio: "Developer advocate and technical writer. Loves Go and open source.", Skills: []string{"Go", "Python", "Technical Writing", "Docker", "Kubernetes"}, Interests: []string{"Developer Experience", "Open Source", "Community Building"}, GithubURL: "https://github.com/emmadevrel", Location: "Austin, USA", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=emma_devrel"},
		{ID: "user011", Username: "raj_systems", Email: "raj@infra.dev", Bio: "Systems programmer contributing to Linux kernel and eBPF projects.", Skills: []string{"C", "Go", "eBPF", "Linux", "Rust"}, Interests: []string{"Operating Systems", "Networking", "Observability"}, GithubURL: "https://github.com/rajsystems", Location: "Hyderabad, India", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=raj_systems"},
		{ID: "user012", Username: "anna_gamedev", Email: "anna@pixelforged.com", Bio: "Game developer working on indie titles with Godot and Unity.", Skills: []string{"C#", "GDScript", "Unity", "Godot", "Blender"}, Interests: []string{"Game Design", "Procedural Generation", "Pixel Art"}, GithubURL: "https://github.com/annagamedev", Location: "Stockholm, Sweden", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=anna_gamedev"},
		{ID: "user013", Username: "chen_embedded", Email: "chen@iotsys.cn", Bio: "Embedded systems engineer working with ARM and RISC-V architectures.", Skills: []string{"C", "C++", "Rust", "RTOS", "ARM"}, Interests: []string{"IoT", "Embedded Linux", "Hardware Design"}, GithubURL: "https://github.com/chenembedded", Location: "Shenzhen, China", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=chen_embedded"},
		{ID: "user014", Username: "fatima_qa", Email: "fatima@testpro.io", Bio: "QA automation engineer building robust test frameworks.", Skills: []string{"Python", "Selenium", "Cypress", "Go", "Docker"}, Interests: []string{"Test Automation", "CI/CD", "Quality Engineering"}, GithubURL: "https://github.com/fatimaqa", Location: "Istanbul, Turkey", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=fatima_qa"},
		{ID: "user015", Username: "alex_platform", Email: "alex@platformeng.io", Bio: "Platform engineer building internal developer platforms with Backstage.", Skills: []string{"Go", "Kubernetes", "Terraform", "Backstage", "ArgoCD"}, Interests: []string{"Platform Engineering", "Developer Productivity", "GitOps"}, GithubURL: "https://github.com/alexplatform", Location: "Amsterdam, Netherlands", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=alex_platform"},
		{ID: "user016", Username: "maya_design", Email: "maya@ux.studio", Bio: "UX engineer bridging design and code. Figma to React specialist.", Skills: []string{"React", "Figma", "CSS", "TypeScript", "Storybook"}, Interests: []string{"Design Systems", "Accessibility", "Motion Design"}, GithubURL: "https://github.com/mayadesign", Location: "Melbourne, Australia", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=maya_design"},
		{ID: "user017", Username: "viktor_db", Email: "viktor@dbscale.io", Bio: "Database engineer specializing in distributed databases and query optimization.", Skills: []string{"PostgreSQL", "CockroachDB", "Go", "SQL", "Redis"}, Interests: []string{"Distributed Databases", "Query Optimization", "CQRS"}, GithubURL: "https://github.com/viktordb", Location: "Warsaw, Poland", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=viktor_db"},
		{ID: "user018", Username: "zara_ai", Email: "zara@mlops.io", Bio: "MLOps engineer deploying AI models at scale with Kubeflow.", Skills: []string{"Python", "Kubernetes", "Docker", "Kubeflow", "Go"}, Interests: []string{"MLOps", "Model Serving", "Feature Stores"}, GithubURL: "https://github.com/zaraai", Location: "Nairobi, Kenya", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=zara_ai"},
		{ID: "user019", Username: "diego_api", Email: "diego@restful.dev", Bio: "API architect designing scalable microservices with Go and gRPC.", Skills: []string{"Go", "gRPC", "REST", "PostgreSQL", "RabbitMQ"}, Interests: []string{"API Design", "Microservices", "Event-Driven Architecture"}, GithubURL: "https://github.com/diegoapi", Location: "Mexico City, Mexico", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=diego_api"},
		{ID: "user020", Username: "sophie_devops", Email: "sophie@cicd.cloud", Bio: "CI/CD specialist building bulletproof deployment pipelines.", Skills: []string{"Jenkins", "GitHub Actions", "Docker", "Kubernetes", "Terraform"}, Interests: []string{"CI/CD", "Release Engineering", "Chaos Engineering"}, GithubURL: "https://github.com/sophiedevops", Location: "Paris, France", AvatarURL: "https://api.dicebear.com/7.x/avataaars/svg?seed=sophie_devops"},
	}

	for _, u := range fakeUsers {
		u.CreatedAt = time.Now().Add(-time.Duration(rand.Intn(90)) * 24 * time.Hour)
		db.Users[u.ID] = u
	}

	// ---------- FAKE PROJECTS ----------
	fakeProjects := []models.Project{
		{ID: "proj001", Title: "Cloud-Native Log Aggregator", Description: "Build a distributed log aggregation service using Go with Kafka as the message bus. Looking for backend devs with streaming experience.", TechStack: []string{"Go", "Apache Kafka", "Docker", "Kubernetes"}, OwnerID: "user001", OwnerName: "arjun_dev", Status: "open", MaxMembers: 4},
		{ID: "proj002", Title: "AI-Powered Code Reviewer", Description: "Develop a GitHub bot that uses LLMs to automatically review PRs and suggest improvements. Need ML and API expertise.", TechStack: []string{"Python", "Go", "FastAPI", "LLMs"}, OwnerID: "user004", OwnerName: "priya_ml", Status: "open", MaxMembers: 3},
		{ID: "proj003", Title: "Real-Time Multiplayer Game Engine", Description: "Creating an indie multiplayer game engine with Godot. Need help with networking and procedural world generation.", TechStack: []string{"GDScript", "Godot", "C#", "WebSocket"}, OwnerID: "user012", OwnerName: "anna_gamedev", Status: "open", MaxMembers: 5},
		{ID: "proj004", Title: "Developer Portfolio Builder", Description: "A self-hosted portfolio site generator that pulls data from GitHub, LinkedIn. Frontend-heavy project.", TechStack: []string{"React", "Next.js", "TypeScript", "Tailwind"}, OwnerID: "user005", OwnerName: "leo_frontend", Status: "open", MaxMembers: 3},
		{ID: "proj005", Title: "Kubernetes Cost Optimizer", Description: "Build a tool that analyzes K8s cluster resource usage and suggests rightsizing. Platform engineering focus.", TechStack: []string{"Go", "Kubernetes", "Prometheus", "Grafana"}, OwnerID: "user015", OwnerName: "alex_platform", Status: "open", MaxMembers: 4},
		{ID: "proj006", Title: "IoT Smart Agriculture System", Description: "Arduino + Raspberry Pi-based precision farming system with real-time sensor data dashboards.", TechStack: []string{"C", "Python", "MQTT", "Grafana", "ARM"}, OwnerID: "user013", OwnerName: "chen_embedded", Status: "open", MaxMembers: 4},
		{ID: "proj007", Title: "Open Source API Gateway", Description: "Build a lightweight, extensible API gateway in Go with rate limiting, auth, and plugin support.", TechStack: []string{"Go", "gRPC", "Redis", "Docker"}, OwnerID: "user019", OwnerName: "diego_api", Status: "in-progress", MaxMembers: 5},
		{ID: "proj008", Title: "DeFi Lending Protocol", Description: "Smart contract-based lending platform on Ethereum. Looking for Solidity and frontend devs.", TechStack: []string{"Solidity", "Hardhat", "React", "TypeScript"}, OwnerID: "user009", OwnerName: "omar_blockchain", Status: "open", MaxMembers: 4},
	}

	for _, p := range fakeProjects {
		p.CreatedAt = time.Now().Add(-time.Duration(rand.Intn(30)) * 24 * time.Hour)
		db.Projects[p.ID] = p
	}

	// ---------- DEFAULT CHAT ROOMS ----------
	defaultRooms := []models.ChatRoom{
		{ID: "room_general", Name: "🌐 General", Description: "Hang out and talk about anything tech-related.", Participants: []string{}},
		{ID: "room_golang", Name: "🐹 Go/Golang", Description: "All things Go — goroutines, channels, and beyond.", Participants: []string{}},
		{ID: "room_devops", Name: "🚀 DevOps & Cloud", Description: "Kubernetes, Docker, Terraform, CI/CD discussions.", Participants: []string{}},
		{ID: "room_ai", Name: "🤖 AI & Machine Learning", Description: "LLMs, RAG, embeddings, and ML engineering.", Participants: []string{}},
		{ID: "room_opensource", Name: "💻 Open Source", Description: "Share your contributions, find projects to contribute to.", Participants: []string{}},
		{ID: "room_career", Name: "💼 Career & Jobs", Description: "Resume reviews, interview prep, job opportunities.", Participants: []string{}},
	}

	for _, r := range defaultRooms {
		db.Rooms[r.ID] = r
	}
}
