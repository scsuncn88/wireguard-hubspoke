🔄 Project Awareness & Context
	•	Always read requirements.md at the start of a new conversation to understand the project’s overall goals, architecture, constraints, and feature set for the WireGuard-based SD-WAN solution.
	•	Check TASK.md before beginning any task. If the task isn’t listed (for example: “Implement Hub failover logic” or “Design Web UI topology view”), add it under “Discovered During Work” with a brief description and today’s date.
	•	Follow naming conventions, directory structure, and architecture patterns defined in requirements.md:
	•	controller/ for control‐plane services
	•	agent/ for node‐side daemon and helpers
	•	ui/ for frontend Web UI
	•	infra/ for Docker/Kubernetes deployment manifests
	•	Activate venv_linux (the project virtual environment) before running any local scripts, tests, or tooling.

🧱 Code Structure & Modularity
	•	Limit any single file to 500 lines. If approaching that, split into feature modules (e.g. agent/config.go, agent/wg_helper.go) or utility packages.
	•	Group code by responsibility:
	•	controller/ hosts the API server, database models, topology manager, and HA logic.
	•	agent/ contains the registration client, config fetcher, and WireGuard wrapper.
	•	ui/ holds frontend React/Vue components, services, and layouts.
	•	common/ provides shared types, constants, and utility functions.
	•	Use clear, consistent imports (prefer relative imports within the repo) and enforce formatting with black or go fmt.
	•	Manage secrets and configuration via python_dotenv (or equivalent) and always call load_env() at startup.

🧪 Testing & Reliability
	•	Every new feature requires unit tests under /tests mirroring the module path:
	•	Controller: test API handlers, topology updates, HA failover.
	•	Agent: test config parsing, WireGuard invocation mocks.
	•	UI: test key components and service calls.
	•	Include three test cases per function: normal behavior, edge case, and failure path.
	•	Run pytest (or go test) against venv_linux before merging any change.
	•	Maintain ≥ 80% code coverage on critical modules (controller and agent).

✅ Task Completion
	•	Immediately mark tasks “Done” in TASK.md when development, code review, and testing are complete.
	•	Log new sub-tasks or bugs discovered during implementation under a “Discovered During Work” section in TASK.md.

📎 Style & Conventions
	•	Primary languages: Go (backend controller & agent) and JavaScript/TypeScript (frontend UI).
	•	Backend: follow idiomatic Go style, use golangci-lint, and apply type‐safe models.
	•	Frontend: use React (or Vue), follow established style guide, enable ESLint and Prettier.
	•	APIs: document with OpenAPI/Swagger, use JSON request/response.
	•	Docstrings/Comments:

// GeneratePeerConfig generates WireGuard config for a spoke node.
// Args:
//   peer: Peer metadata (name, public key, allowed IPs).
// Returns:
//   wgConf: WireGuard configuration block as string.
func GeneratePeerConfig(peer Peer) (wgConf string, err error) { ... }

/**
 * Fetches updated topology from the controller.
 * @returns {Promise<Topology>}
 */
async function fetchTopology() { ... }



📚 Documentation & Explainability
	•	Update README.md whenever setup steps, dependencies, or start-up commands change.
	•	Document API changes in docs/api/ (OpenAPI YAML or markdown).
	•	Comment non-trivial logic and include # Reason: to explain design decisions, especially for HA and failover code.

🧠 AI Behavior Rules
	•	Never assume missing technical context. If a requirement is unclear (e.g. fallback order for multiple Hubs), ask a clarifying question.
	•	Do not hallucinate functions or libraries. Only use vetted, widely adopted packages (e.g. wgctrl for Go, axios for JS).
	•	Always verify file paths and module names before referencing them in code samples or tests.
	•	Do not delete or overwrite existing code unless explicitly instructed by a task in TASK.md or by a maintainer.