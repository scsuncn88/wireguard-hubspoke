FEATURE
	•	基于 WireGuard 的 Hub-and-Spoke SD-WAN 管理系统
	•	Controller（控制平面）：提供集中式 RESTful API 和 Web UI 后端，负责节点注册、拓扑管理、WireGuard 配置自动生成与分发
	•	Agent（边缘守护进程）：自动注册到控制器，拉取并应用 WireGuard 配置，管理加密隧道并上报状态
	•	Web UI（可视化界面）：动态图形化网络拓扑展示、策略与路由可视化编辑、实时监控面板、操作审计日志
	•	CLI（命令行工具）：脚本化运维与快速调试接口
	•	Infra（部署模板）：Docker Compose、Helm Chart、Ansible 等一键部署方案，兼容 Ubuntu 和容器化环境
	•	Common（通用模块）：密钥对管理、配置文件解析、日志与错误处理、参数校验等公共库

REFERENCE PROJECTS

本项目参考现有 WireGuard Mesh 配置工具和 SD-WAN 平台：
	•	wg-meshconf/ — 原项目目录，提供批量生成 WireGuard 全网 Mesh 配置的示例代码，可作为配置生成逻辑参考。
	•	netmaker/（可选）— Netmaker 开源 SD-WAN 平台，示例其集中式控制器与 Agent 架构、ACL 与动态拓扑管理。

注意：上述目录为参考，不要直接复制其中代码。根据本项目的 Hub-and-Spoke 需求，设计并实现更完整的控制平面与分布式 Agent 架构。

DOCUMENTATION
	•	WireGuard 官方文档：https://www.wireguard.com/
	•	wgctrl-go（Go 语言 WireGuard 控制库）：https://github.com/WireGuard/wgctrl-go
	•	Netmaker 开源 SD-WAN 平台：https://github.com/gravitl/netmaker
	•	Prometheus 监控系统：https://prometheus.io/
	•	React 前端框架：https://reactjs.org/
	•	Docker 官方文档：https://docs.docker.com/

OTHER CONSIDERATIONS
	•	项目根目录应包含：requirements.md、PLANNING.md、TASK.md 和 .env.example
	•	在 README 中说明项目结构、环境变量配置（如 CONTROLLER_URL、DB_CONN、JWT_SECRET 等）、部署与使用步骤
	•	使用 venv_linux 虚拟环境，应用入口启动时调用 load_env() 加载环境变量
	•	代码风格：
	•	Go 代码：启用 golangci-lint，自动运行 go fmt
	•	前端：启用 ESLint 与 Prettier
	•	Python 脚本：使用 black 格式化
	•	提供 Docker Compose 与 Helm Chart 示例以实现一键部署
	•	确保 API 文档（OpenAPI/Swagger）与实际接口一致，并在 docs/api/ 中维护