# WireGuard SD-WAN 部署和使用手册

## 📋 目录
1. [系统概览](#系统概览)
2. [环境要求](#环境要求)
3. [安装部署](#安装部署)
4. [配置说明](#配置说明)
5. [使用指南](#使用指南)
6. [API接口](#api接口)
7. [监控运维](#监控运维)
8. [故障排除](#故障排除)
9. [安全最佳实践](#安全最佳实践)
10. [常见问题](#常见问题)

---

## 🎯 系统概览

WireGuard SD-WAN是一个基于WireGuard的企业级软件定义广域网解决方案，提供：

### 核心功能
- **节点管理**: Hub和Spoke节点自动注册和配置
- **用户认证**: 基于JWT的身份验证和RBAC权限控制
- **网络配置**: 自动生成WireGuard配置文件
- **监控审计**: 实时监控和完整的审计日志
- **高可用性**: 支持控制器集群和故障转移
- **安全加固**: 多层安全防护和策略管理

### 系统架构
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Controller    │    │   Controller    │    │      Web UI     │
│   (Primary)     │────│   (Backup)      │────│                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
    ┌────────────────────────────┼────────────────────────────┐
    │                            │                            │
┌─────────┐                 ┌─────────┐                 ┌─────────┐
│Hub Node │                 │Hub Node │                 │Spoke    │
│(Main)   │─────────────────│(Backup) │─────────────────│Nodes    │
└─────────┘                 └─────────┘                 └─────────┘
```

---

## 🔧 环境要求

### 硬件要求

#### 控制器服务器
- **CPU**: 最低2核，推荐4核+
- **内存**: 最低2GB，推荐4GB+
- **存储**: 最低20GB SSD，推荐50GB+
- **网络**: 1Gbps网络接口

#### Hub节点
- **CPU**: 最低2核，推荐4核+
- **内存**: 最低1GB，推荐2GB+
- **存储**: 最低10GB，推荐20GB+
- **网络**: 稳定的公网IP地址

#### Spoke节点
- **CPU**: 最低1核，推荐2核+
- **内存**: 最低512MB，推荐1GB+
- **存储**: 最低5GB，推荐10GB+
- **网络**: 互联网连接

### 软件要求

#### 操作系统
- **Linux**: Ubuntu 20.04+, CentOS 8+, RHEL 8+
- **macOS**: macOS 11.0+ (开发测试)
- **Windows**: Windows Server 2019+ (实验性支持)

#### 依赖软件
- **Go**: 1.21+
- **PostgreSQL**: 13.0+
- **WireGuard**: 1.0+
- **Docker**: 20.10+ (可选)
- **Kubernetes**: 1.24+ (可选)

---

## 🚀 安装部署

### 方式一：二进制部署

#### 1. 下载二进制文件
```bash
# 下载最新版本
wget https://github.com/wg-hubspoke/wg-hubspoke/releases/latest/download/wg-hubspoke-linux-amd64.tar.gz

# 解压
tar -xzf wg-hubspoke-linux-amd64.tar.gz
cd wg-hubspoke

# 设置权限
chmod +x controller/controller
chmod +x agent/agent
```

#### 2. 安装WireGuard
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install wireguard

# CentOS/RHEL
sudo yum install epel-release
sudo yum install wireguard-tools

# 验证安装
wg version
```

#### 3. 安装PostgreSQL
```bash
# Ubuntu/Debian
sudo apt install postgresql postgresql-contrib

# CentOS/RHEL
sudo yum install postgresql-server postgresql-contrib

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 创建数据库
sudo -u postgres createdb wg_sdwan
sudo -u postgres createuser wg_user
sudo -u postgres psql -c "ALTER USER wg_user WITH PASSWORD 'secure_password';"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE wg_sdwan TO wg_user;"
```

#### 4. 配置环境变量
```bash
# 复制环境变量模板
cp .env.example .env

# 编辑配置文件
nano .env
```

#### 5. 启动控制器
```bash
# 初始化数据库
./controller/controller --migrate

# 启动服务
./controller/controller --config=config/controller.yaml
```

### 方式二：Docker部署

#### 1. 使用Docker Compose
```yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: wg_sdwan
      POSTGRES_USER: wg_user
      POSTGRES_PASSWORD: secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  controller:
    image: wg-hubspoke/controller:latest
    depends_on:
      - postgres
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=wg_sdwan
      - DB_USER=wg_user
      - DB_PASSWORD=secure_password
    ports:
      - "8080:8080"
    volumes:
      - ./config:/app/config
      - controller_data:/app/data

  web-ui:
    image: wg-hubspoke/web-ui:latest
    depends_on:
      - controller
    environment:
      - API_URL=http://controller:8080
    ports:
      - "3000:3000"

volumes:
  postgres_data:
  controller_data:
```

```bash
# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f controller
```

### 方式三：Kubernetes部署

#### 1. 创建命名空间
```bash
kubectl create namespace wg-sdwan
```

#### 2. 部署PostgreSQL
```yaml
# postgres-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: wg-sdwan
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15
        env:
        - name: POSTGRES_DB
          value: wg_sdwan
        - name: POSTGRES_USER
          value: wg_user
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
```

#### 3. 部署控制器
```yaml
# controller-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller
  namespace: wg-sdwan
spec:
  replicas: 2
  selector:
    matchLabels:
      app: controller
  template:
    metadata:
      labels:
        app: controller
    spec:
      containers:
      - name: controller
        image: wg-hubspoke/controller:latest
        env:
        - name: DB_HOST
          value: postgres-service
        - name: DB_PORT
          value: "5432"
        - name: DB_NAME
          value: wg_sdwan
        - name: DB_USER
          value: wg_user
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

```bash
# 部署应用
kubectl apply -f postgres-deployment.yaml
kubectl apply -f controller-deployment.yaml

# 查看状态
kubectl get pods -n wg-sdwan
kubectl get services -n wg-sdwan
```

---

## ⚙️ 配置说明

### 控制器配置文件

#### config/controller.yaml
```yaml
# 服务配置
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

# 数据库配置
database:
  host: "localhost"
  port: 5432
  name: "wg_sdwan"
  user: "wg_user"
  password: "secure_password"
  sslmode: "require"
  max_connections: 100
  max_idle_connections: 10
  connection_lifetime: 300s

# JWT配置
jwt:
  secret: "your-jwt-secret-key-here"
  expires_in: 24h
  refresh_expires_in: 168h

# WireGuard配置
wireguard:
  interface: "wg0"
  port: 51820
  subnet: "10.100.0.0/16"
  dns_servers:
    - "8.8.8.8"
    - "8.8.4.4"

# 监控配置
monitoring:
  enabled: true
  metrics_port: 9090
  health_check_interval: 30s
  prometheus_endpoint: "/metrics"

# 审计配置
audit:
  enabled: true
  log_level: "info"
  retention_days: 90
  max_log_size: "100MB"

# 安全配置
security:
  max_login_attempts: 5
  lockout_duration: 15m
  password_min_length: 8
  password_require_special: true
  rate_limit:
    enabled: true
    requests_per_minute: 60
    burst_size: 100

# 高可用配置
ha:
  enabled: false
  cluster_id: "wg-sdwan-cluster"
  election_timeout: 5s
  heartbeat_interval: 1s
  nodes:
    - "controller-1:8080"
    - "controller-2:8080"

# 备份配置
backup:
  enabled: true
  schedule: "0 2 * * *"  # 每天凌晨2点
  retention_days: 30
  storage_path: "/var/lib/wg-sdwan/backups"
  compression: true
```

### 环境变量配置

#### .env文件
```bash
# 基础配置
GO_ENV=production
LOG_LEVEL=info
DEBUG=false

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_NAME=wg_sdwan
DB_USER=wg_user
DB_PASSWORD=secure_password
DB_SSLMODE=require

# JWT配置
JWT_SECRET=your-very-secure-jwt-secret-key-here
JWT_EXPIRES_IN=24h

# WireGuard配置
WG_INTERFACE=wg0
WG_PORT=51820
WG_SUBNET=10.100.0.0/16
WG_DNS=8.8.8.8,8.8.4.4

# 控制器配置
CONTROLLER_HOST=0.0.0.0
CONTROLLER_PORT=8080
CONTROLLER_TLS_ENABLED=true
CONTROLLER_TLS_CERT=/etc/ssl/certs/wg-sdwan.crt
CONTROLLER_TLS_KEY=/etc/ssl/private/wg-sdwan.key

# 监控配置
MONITORING_ENABLED=true
METRICS_PORT=9090
PROMETHEUS_ENABLED=true

# 安全配置
SECURITY_ENABLED=true
MAX_LOGIN_ATTEMPTS=5
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPM=60

# 高可用配置
HA_ENABLED=false
HA_CLUSTER_ID=wg-sdwan-cluster
HA_NODES=controller-1:8080,controller-2:8080

# 备份配置
BACKUP_ENABLED=true
BACKUP_SCHEDULE=0 2 * * *
BACKUP_RETENTION_DAYS=30
BACKUP_PATH=/var/lib/wg-sdwan/backups
```

### 代理配置

#### nginx.conf
```nginx
upstream wg_sdwan_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081 backup;
}

server {
    listen 80;
    server_name wg-sdwan.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name wg-sdwan.example.com;

    ssl_certificate /etc/ssl/certs/wg-sdwan.crt;
    ssl_certificate_key /etc/ssl/private/wg-sdwan.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    client_max_body_size 10M;

    # API代理
    location /api/ {
        proxy_pass http://wg_sdwan_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    # Web UI
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket支持
    location /ws/ {
        proxy_pass http://wg_sdwan_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 健康检查
    location /health {
        proxy_pass http://wg_sdwan_backend;
        access_log off;
    }

    # 指标端点
    location /metrics {
        proxy_pass http://wg_sdwan_backend;
        allow 127.0.0.1;
        allow 10.0.0.0/8;
        deny all;
    }
}
```

---

## 📖 使用指南

### 初始化系统

#### 1. 创建管理员账户
```bash
# 使用命令行创建管理员
./controller/controller --create-admin \
  --username=admin \
  --email=admin@example.com \
  --password=SecurePassword123!

# 或者使用API
curl -X POST https://wg-sdwan.example.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "SecurePassword123!",
    "role": "admin"
  }'
```

#### 2. 登录系统
```bash
# 获取访问令牌
curl -X POST https://wg-sdwan.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "SecurePassword123!"
  }'
```

### 节点管理

#### 1. 注册Hub节点
```bash
# 通过API注册Hub节点
curl -X POST https://wg-sdwan.example.com/api/v1/nodes \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hub-main",
    "node_type": "hub",
    "public_key": "YOUR_HUB_PUBLIC_KEY",
    "endpoint": "hub.example.com:51820",
    "description": "Main hub node"
  }'
```

#### 2. 注册Spoke节点
```bash
# 通过API注册Spoke节点
curl -X POST https://wg-sdwan.example.com/api/v1/nodes \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "spoke-branch1",
    "node_type": "spoke",
    "public_key": "YOUR_SPOKE_PUBLIC_KEY",
    "description": "Branch office 1"
  }'
```

#### 3. 获取节点配置
```bash
# 获取节点的WireGuard配置
curl -X GET https://wg-sdwan.example.com/api/v1/nodes/NODE_ID/config \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 代理部署

#### 1. 在节点上安装代理
```bash
# 下载并安装代理
wget https://github.com/wg-hubspoke/wg-hubspoke/releases/latest/download/agent-linux-amd64.tar.gz
tar -xzf agent-linux-amd64.tar.gz
sudo cp agent /usr/local/bin/

# 创建配置文件
sudo mkdir -p /etc/wg-sdwan
sudo tee /etc/wg-sdwan/agent.yaml > /dev/null <<EOF
controller:
  url: "https://wg-sdwan.example.com"
  api_key: "YOUR_API_KEY"
  
node:
  name: "node-name"
  type: "spoke"
  
wireguard:
  interface: "wg0"
  config_path: "/etc/wireguard/wg0.conf"
  
monitoring:
  enabled: true
  interval: 30s
  
logging:
  level: "info"
  file: "/var/log/wg-sdwan/agent.log"
EOF
```

#### 2. 创建系统服务
```bash
# 创建systemd服务文件
sudo tee /etc/systemd/system/wg-sdwan-agent.service > /dev/null <<EOF
[Unit]
Description=WireGuard SD-WAN Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/agent --config=/etc/wg-sdwan/agent.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# 启用并启动服务
sudo systemctl daemon-reload
sudo systemctl enable wg-sdwan-agent
sudo systemctl start wg-sdwan-agent

# 查看状态
sudo systemctl status wg-sdwan-agent
```

### Web UI使用

#### 1. 访问Web界面
打开浏览器访问: `https://wg-sdwan.example.com`

#### 2. 主要功能

**仪表板**
- 系统概览和统计信息
- 节点状态和连接图
- 实时监控指标
- 告警和通知

**节点管理**
- 查看所有节点列表
- 添加/编辑/删除节点
- 生成和下载配置文件
- 节点状态监控

**用户管理**
- 用户账户管理
- 角色权限分配
- 访问日志查看
- 安全策略设置

**监控中心**
- 实时性能监控
- 网络流量分析
- 连接状态监控
- 历史数据查看

**审计日志**
- 操作日志查看
- 安全事件监控
- 日志搜索和过滤
- 合规性报告

**系统设置**
- 基础配置管理
- 安全策略设置
- 备份和恢复
- 系统维护

---

## 🔌 API接口

### 认证接口

#### 用户登录
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password123"
}
```

响应:
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 86400,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin"
    }
  }
}
```

#### 刷新令牌
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### 节点管理接口

#### 获取节点列表
```http
GET /api/v1/nodes?page=1&per_page=20&node_type=hub
Authorization: Bearer YOUR_TOKEN
```

响应:
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "hub-main",
      "node_type": "hub",
      "public_key": "ABC123...",
      "allocated_ip": "10.100.1.1",
      "endpoint": "hub.example.com:51820",
      "status": "active",
      "last_seen": "2024-01-15T10:30:00Z",
      "created_at": "2024-01-15T09:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

#### 创建节点
```http
POST /api/v1/nodes
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "name": "hub-backup",
  "node_type": "hub",
  "public_key": "XYZ789...",
  "endpoint": "hub-backup.example.com:51820",
  "description": "Backup hub node"
}
```

#### 获取节点配置
```http
GET /api/v1/nodes/550e8400-e29b-41d4-a716-446655440000/config
Authorization: Bearer YOUR_TOKEN
```

响应:
```json
{
  "success": true,
  "data": {
    "config": "[Interface]\nPrivateKey = [PRIVATE_KEY]\nAddress = 10.100.1.1/24\nListenPort = 51820\n\n[Peer]\nPublicKey = ABC123...\nAllowedIPs = 10.100.0.0/16\nEndpoint = hub.example.com:51820\nPersistentKeepalive = 25"
  }
}
```

### 监控接口

#### 更新节点指标
```http
POST /api/v1/monitoring/nodes/550e8400-e29b-41d4-a716-446655440000/metrics
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "cpu_usage": 25.5,
  "memory_usage": 60.2,
  "disk_usage": 45.8,
  "network_rx": 1024000,
  "network_tx": 512000,
  "wireguard_peers": 5,
  "active_connections": 12
}
```

#### 获取节点指标
```http
GET /api/v1/monitoring/nodes/550e8400-e29b-41d4-a716-446655440000/metrics
Authorization: Bearer YOUR_TOKEN
```

### 用户管理接口

#### 创建用户
```http
POST /api/v1/users
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "username": "operator",
  "email": "operator@example.com",
  "password": "SecurePassword123!",
  "role": "user"
}
```

#### 获取用户列表
```http
GET /api/v1/users?page=1&per_page=20
Authorization: Bearer YOUR_TOKEN
```

### 审计日志接口

#### 获取审计日志
```http
GET /api/v1/audit/logs?page=1&per_page=50&action=CREATE_NODE
Authorization: Bearer YOUR_TOKEN
```

响应:
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "550e8400-e29b-41d4-a716-446655440001",
      "action": "CREATE_NODE",
      "resource": "nodes",
      "resource_id": "550e8400-e29b-41d4-a716-446655440002",
      "details": {
        "node_name": "hub-main",
        "node_type": "hub"
      },
      "ip_address": "192.168.1.100",
      "user_agent": "Mozilla/5.0...",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 50,
    "total": 1,
    "total_pages": 1
  }
}
```

### 系统接口

#### 健康检查
```http
GET /health
```

响应:
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "version": "1.0.0",
  "checks": {
    "database": "healthy",
    "redis": "healthy",
    "wireguard": "healthy"
  }
}
```

#### Prometheus指标
```http
GET /metrics
```

---

## 📊 监控运维

### Prometheus监控

#### 1. 配置Prometheus
```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'wg-sdwan-controller'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: '/metrics'
    scrape_interval: 30s

  - job_name: 'wg-sdwan-nodes'
    consul_sd_configs:
      - server: 'localhost:8500'
        services: ['wg-sdwan-node']
    relabel_configs:
      - source_labels: [__meta_consul_service_address]
        target_label: __address__
        replacement: '${1}:9090'
```

#### 2. 关键指标

**系统指标**
- `wg_sdwan_nodes_total{type="hub|spoke"}` - 节点总数
- `wg_sdwan_nodes_active{type="hub|spoke"}` - 活跃节点数
- `wg_sdwan_connections_total` - 连接总数
- `wg_sdwan_traffic_bytes{direction="rx|tx"}` - 流量统计

**性能指标**
- `wg_sdwan_cpu_usage_percent` - CPU使用率
- `wg_sdwan_memory_usage_percent` - 内存使用率
- `wg_sdwan_disk_usage_percent` - 磁盘使用率
- `wg_sdwan_network_latency_ms` - 网络延迟

**安全指标**
- `wg_sdwan_login_attempts_total{result="success|failure"}` - 登录尝试
- `wg_sdwan_security_events_total{type="blocked_ip|failed_login"}` - 安全事件
- `wg_sdwan_rate_limit_exceeded_total` - 速率限制触发

### Grafana仪表板

#### 1. 导入仪表板
```json
{
  "dashboard": {
    "title": "WireGuard SD-WAN Overview",
    "panels": [
      {
        "title": "Active Nodes",
        "type": "stat",
        "targets": [
          {
            "expr": "wg_sdwan_nodes_active",
            "legendFormat": "{{type}}"
          }
        ]
      },
      {
        "title": "Network Traffic",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(wg_sdwan_traffic_bytes[5m])",
            "legendFormat": "{{direction}}"
          }
        ]
      },
      {
        "title": "Connection Status",
        "type": "table",
        "targets": [
          {
            "expr": "wg_sdwan_connections_total",
            "format": "table"
          }
        ]
      }
    ]
  }
}
```

### 日志管理

#### 1. 配置日志聚合
```yaml
# filebeat.yml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/wg-sdwan/*.log
  fields:
    service: wg-sdwan
    environment: production
  fields_under_root: true

output.elasticsearch:
  hosts: ["localhost:9200"]
  index: "wg-sdwan-logs-%{+yyyy.MM.dd}"

logging.level: info
logging.to_files: true
logging.files:
  path: /var/log/filebeat
  name: filebeat
  keepfiles: 7
  permissions: 0644
```

#### 2. 日志查询示例
```bash
# 查看错误日志
grep -i "error" /var/log/wg-sdwan/controller.log

# 查看登录失败
grep "login failed" /var/log/wg-sdwan/controller.log

# 查看节点连接
grep "node connected" /var/log/wg-sdwan/controller.log

# 使用journalctl查看服务日志
journalctl -u wg-sdwan-controller -f
```

### 告警配置

#### 1. Alertmanager配置
```yaml
# alertmanager.yml
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@example.com'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'

receivers:
- name: 'default'
  email_configs:
  - to: 'admin@example.com'
    subject: 'WireGuard SD-WAN Alert'
    body: |
      {{ range .Alerts }}
      Alert: {{ .Annotations.summary }}
      Description: {{ .Annotations.description }}
      {{ end }}
```

#### 2. 告警规则
```yaml
# alert-rules.yml
groups:
- name: wg-sdwan
  rules:
  - alert: NodeDown
    expr: wg_sdwan_nodes_active < 1
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "WireGuard SD-WAN node is down"
      description: "Node {{ $labels.node_name }} has been down for more than 5 minutes"

  - alert: HighCPUUsage
    expr: wg_sdwan_cpu_usage_percent > 90
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High CPU usage detected"
      description: "CPU usage is above 90% for more than 2 minutes"

  - alert: LoginFailures
    expr: increase(wg_sdwan_login_attempts_total{result="failure"}[5m]) > 10
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "Multiple login failures detected"
      description: "More than 10 login failures in the last 5 minutes"
```

### 备份策略

#### 1. 数据库备份
```bash
#!/bin/bash
# backup-database.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/backups/wg-sdwan"
DB_NAME="wg_sdwan"

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 备份数据库
pg_dump -h localhost -U wg_user -d "$DB_NAME" > "$BACKUP_DIR/db_backup_$DATE.sql"

# 压缩备份
gzip "$BACKUP_DIR/db_backup_$DATE.sql"

# 删除7天前的备份
find "$BACKUP_DIR" -name "db_backup_*.sql.gz" -mtime +7 -delete

echo "Database backup completed: $BACKUP_DIR/db_backup_$DATE.sql.gz"
```

#### 2. 配置备份
```bash
#!/bin/bash
# backup-config.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/backups/wg-sdwan"
CONFIG_DIR="/etc/wg-sdwan"

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 备份配置文件
tar -czf "$BACKUP_DIR/config_backup_$DATE.tar.gz" "$CONFIG_DIR"

# 删除30天前的备份
find "$BACKUP_DIR" -name "config_backup_*.tar.gz" -mtime +30 -delete

echo "Configuration backup completed: $BACKUP_DIR/config_backup_$DATE.tar.gz"
```

#### 3. 自动备份
```bash
# 添加到crontab
crontab -e

# 每天2点备份数据库
0 2 * * * /usr/local/bin/backup-database.sh

# 每天3点备份配置
0 3 * * * /usr/local/bin/backup-config.sh
```

---

## 🛠️ 故障排除

### 常见问题诊断

#### 1. 控制器无法启动
```bash
# 检查服务状态
systemctl status wg-sdwan-controller

# 查看日志
journalctl -u wg-sdwan-controller -f

# 检查配置文件
./controller/controller --config-check

# 检查数据库连接
./controller/controller --db-check
```

可能原因和解决方案:
- **数据库连接失败**: 检查数据库配置和网络连接
- **端口被占用**: 更改配置文件中的端口设置
- **权限问题**: 确保服务有足够的文件和网络权限
- **配置错误**: 验证配置文件格式和内容

#### 2. 节点无法连接
```bash
# 检查节点状态
wg show

# 检查WireGuard配置
wg showconf wg0

# 测试网络连接
ping 10.100.1.1

# 检查防火墙
iptables -L -n
ufw status
```

可能原因和解决方案:
- **防火墙阻塞**: 开放WireGuard端口(默认51820)
- **NAT问题**: 配置端口转发或使用STUN
- **密钥错误**: 重新生成和配置密钥对
- **路由问题**: 检查和修正路由表

#### 3. 认证失败
```bash
# 检查JWT配置
grep JWT /etc/wg-sdwan/controller.yaml

# 测试API连接
curl -v https://wg-sdwan.example.com/api/v1/health

# 检查证书
openssl s_client -connect wg-sdwan.example.com:443
```

可能原因和解决方案:
- **令牌过期**: 重新登录获取新令牌
- **时间同步**: 确保系统时间正确
- **证书问题**: 更新SSL证书
- **密钥配置**: 检查JWT密钥配置

#### 4. 性能问题
```bash
# 检查系统资源
top
htop
iostat -x 1

# 检查网络状态
netstat -i
ss -tuln

# 检查数据库性能
psql -U wg_user -d wg_sdwan -c "SELECT * FROM pg_stat_activity;"
```

优化建议:
- **数据库优化**: 添加索引、调整连接池
- **内存优化**: 增加系统内存或调整应用配置
- **网络优化**: 调整网络参数和MTU设置
- **负载均衡**: 部署多个控制器实例

### 诊断命令

#### 1. 系统诊断
```bash
# 创建诊断脚本
cat > /usr/local/bin/wg-sdwan-diag.sh << 'EOF'
#!/bin/bash
echo "=== WireGuard SD-WAN Diagnostics ==="
echo "Date: $(date)"
echo "Hostname: $(hostname)"
echo "OS: $(cat /etc/os-release | grep PRETTY_NAME)"
echo ""

echo "=== Service Status ==="
systemctl status wg-sdwan-controller
echo ""

echo "=== Network Status ==="
ip addr show
echo ""

echo "=== WireGuard Status ==="
wg show
echo ""

echo "=== Database Status ==="
pg_isready -h localhost -p 5432
echo ""

echo "=== Disk Usage ==="
df -h
echo ""

echo "=== Memory Usage ==="
free -h
echo ""

echo "=== CPU Usage ==="
top -bn1 | grep "Cpu(s)"
echo ""

echo "=== Recent Logs ==="
journalctl -u wg-sdwan-controller --since "1 hour ago" --no-pager
EOF

chmod +x /usr/local/bin/wg-sdwan-diag.sh
```

#### 2. 网络诊断
```bash
# 网络连接测试
cat > /usr/local/bin/wg-sdwan-netcheck.sh << 'EOF'
#!/bin/bash
echo "=== Network Connectivity Check ==="

# 检查DNS解析
echo "DNS Resolution:"
nslookup wg-sdwan.example.com
echo ""

# 检查端口连接
echo "Port Connectivity:"
nc -zv wg-sdwan.example.com 443
nc -zv wg-sdwan.example.com 51820
echo ""

# 检查WireGuard接口
echo "WireGuard Interface:"
ip addr show wg0
echo ""

# 检查路由
echo "Routing Table:"
ip route show
echo ""

# 检查防火墙
echo "Firewall Rules:"
iptables -L -n | grep -E "(51820|8080|443)"
EOF

chmod +x /usr/local/bin/wg-sdwan-netcheck.sh
```

### 恢复程序

#### 1. 数据库恢复
```bash
# 从备份恢复数据库
./restore-database.sh backup_file.sql.gz

# 恢复脚本
cat > /usr/local/bin/restore-database.sh << 'EOF'
#!/bin/bash
if [ $# -ne 1 ]; then
    echo "Usage: $0 <backup_file.sql.gz>"
    exit 1
fi

BACKUP_FILE=$1
DB_NAME="wg_sdwan"

echo "Restoring database from $BACKUP_FILE..."

# 停止服务
systemctl stop wg-sdwan-controller

# 删除现有数据库
sudo -u postgres dropdb "$DB_NAME"

# 创建新数据库
sudo -u postgres createdb "$DB_NAME"

# 恢复数据
gunzip -c "$BACKUP_FILE" | sudo -u postgres psql -d "$DB_NAME"

# 启动服务
systemctl start wg-sdwan-controller

echo "Database restoration completed"
EOF

chmod +x /usr/local/bin/restore-database.sh
```

#### 2. 配置恢复
```bash
# 从备份恢复配置
./restore-config.sh config_backup.tar.gz

# 恢复脚本
cat > /usr/local/bin/restore-config.sh << 'EOF'
#!/bin/bash
if [ $# -ne 1 ]; then
    echo "Usage: $0 <config_backup.tar.gz>"
    exit 1
fi

BACKUP_FILE=$1
CONFIG_DIR="/etc/wg-sdwan"

echo "Restoring configuration from $BACKUP_FILE..."

# 备份当前配置
cp -r "$CONFIG_DIR" "$CONFIG_DIR.bak.$(date +%Y%m%d_%H%M%S)"

# 恢复配置
tar -xzf "$BACKUP_FILE" -C /

# 重新加载配置
systemctl reload wg-sdwan-controller

echo "Configuration restoration completed"
EOF

chmod +x /usr/local/bin/restore-config.sh
```

### 性能调优

#### 1. 系统调优
```bash
# 创建调优脚本
cat > /usr/local/bin/wg-sdwan-tune.sh << 'EOF'
#!/bin/bash
echo "=== WireGuard SD-WAN Performance Tuning ==="

# 网络参数优化
echo "Optimizing network parameters..."
sysctl -w net.core.rmem_max=134217728
sysctl -w net.core.wmem_max=134217728
sysctl -w net.ipv4.tcp_rmem="4096 87380 134217728"
sysctl -w net.ipv4.tcp_wmem="4096 65536 134217728"
sysctl -w net.ipv4.tcp_congestion_control=bbr

# 文件描述符限制
echo "Increasing file descriptor limits..."
echo "* soft nofile 65535" >> /etc/security/limits.conf
echo "* hard nofile 65535" >> /etc/security/limits.conf

# 内核参数优化
echo "Optimizing kernel parameters..."
echo "net.core.netdev_max_backlog = 5000" >> /etc/sysctl.conf
echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.conf
sysctl -p

echo "Performance tuning completed"
EOF

chmod +x /usr/local/bin/wg-sdwan-tune.sh
```

#### 2. 数据库调优
```bash
# PostgreSQL调优
cat > /etc/postgresql/15/main/postgresql.conf.d/wg-sdwan.conf << 'EOF'
# WireGuard SD-WAN PostgreSQL Configuration

# 内存设置
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB
maintenance_work_mem = 64MB

# 连接设置
max_connections = 200
superuser_reserved_connections = 3

# 写入性能
wal_buffers = 16MB
checkpoint_completion_target = 0.7
checkpoint_timeout = 10min
max_wal_size = 2GB
min_wal_size = 1GB

# 查询优化
random_page_cost = 1.1
effective_io_concurrency = 200

# 日志设置
log_min_duration_statement = 1000
log_checkpoints = on
log_connections = on
log_disconnections = on
log_lock_waits = on
log_temp_files = 0

# 统计设置
track_activities = on
track_counts = on
track_io_timing = on
track_functions = pl
EOF

# 重启PostgreSQL
systemctl restart postgresql
```

---

## 🔒 安全最佳实践

### 系统安全

#### 1. 操作系统加固
```bash
# 更新系统
apt update && apt upgrade -y

# 安装安全工具
apt install -y fail2ban ufw lynis rkhunter chkrootkit

# 配置防火墙
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp
ufw allow 443/tcp
ufw allow 51820/udp
ufw enable

# 配置Fail2Ban
cat > /etc/fail2ban/jail.local << 'EOF'
[sshd]
enabled = true
port = 22
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
bantime = 3600

[wg-sdwan]
enabled = true
port = 443
filter = wg-sdwan
logpath = /var/log/wg-sdwan/controller.log
maxretry = 5
bantime = 1800
EOF

# 创建自定义过滤器
cat > /etc/fail2ban/filter.d/wg-sdwan.conf << 'EOF'
[Definition]
failregex = ^.*login failed.*from <HOST>.*$
ignoreregex =
EOF

# 重启服务
systemctl restart fail2ban
```

#### 2. SSH安全配置
```bash
# 配置SSH
cat > /etc/ssh/sshd_config.d/wg-sdwan.conf << 'EOF'
# WireGuard SD-WAN SSH Configuration
Port 22
Protocol 2
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
AuthorizedKeysFile .ssh/authorized_keys
MaxAuthTries 3
MaxSessions 10
ClientAliveInterval 300
ClientAliveCountMax 2
X11Forwarding no
AllowTcpForwarding no
GatewayPorts no
PermitTunnel no
EOF

# 重启SSH服务
systemctl restart sshd
```

### 应用安全

#### 1. SSL/TLS配置
```bash
# 生成证书签名请求
openssl req -new -newkey rsa:4096 -nodes -keyout wg-sdwan.key -out wg-sdwan.csr -subj "/C=US/ST=State/L=City/O=Organization/CN=wg-sdwan.example.com"

# 或使用Let's Encrypt
certbot certonly --standalone -d wg-sdwan.example.com

# 配置强加密
cat > /etc/nginx/conf.d/ssl.conf << 'EOF'
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-CHACHA20-POLY1305;
ssl_prefer_server_ciphers off;
ssl_session_cache shared:SSL:10m;
ssl_session_timeout 1d;
ssl_stapling on;
ssl_stapling_verify on;
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
add_header X-Frame-Options DENY always;
add_header X-Content-Type-Options nosniff always;
add_header X-XSS-Protection "1; mode=block" always;
EOF
```

#### 2. 数据库安全
```bash
# 配置PostgreSQL安全
cat > /etc/postgresql/15/main/conf.d/security.conf << 'EOF'
# 连接安全
ssl = on
ssl_cert_file = '/etc/ssl/certs/postgresql.crt'
ssl_key_file = '/etc/ssl/private/postgresql.key'
ssl_ca_file = '/etc/ssl/certs/ca-certificates.crt'

# 认证安全
password_encryption = scram-sha-256
row_security = on

# 审计日志
log_connections = on
log_disconnections = on
log_statement = 'all'
log_min_duration_statement = 0
EOF

# 配置访问控制
cat > /etc/postgresql/15/main/pg_hba.conf << 'EOF'
# TYPE  DATABASE        USER            ADDRESS                 METHOD
local   all             postgres                                peer
local   all             all                                     peer
hostssl wg_sdwan        wg_user         127.0.0.1/32            scram-sha-256
hostssl wg_sdwan        wg_user         ::1/128                 scram-sha-256
EOF

# 重启PostgreSQL
systemctl restart postgresql
```

### 网络安全

#### 1. WireGuard安全配置
```bash
# 生成安全的密钥
wg genkey | tee private.key | wg pubkey > public.key

# 配置严格的防火墙规则
cat > /etc/wireguard/wg0.conf << 'EOF'
[Interface]
PrivateKey = PRIVATE_KEY_HERE
Address = 10.100.1.1/24
ListenPort = 51820
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE
SaveConfig = true

[Peer]
PublicKey = PEER_PUBLIC_KEY_HERE
AllowedIPs = 10.100.2.0/24
PersistentKeepalive = 25
EOF

# 设置安全权限
chmod 600 /etc/wireguard/wg0.conf
chown root:root /etc/wireguard/wg0.conf
```

#### 2. 网络分段
```bash
# 创建网络分段规则
cat > /etc/iptables/rules.v4 << 'EOF'
*filter
:INPUT ACCEPT [0:0]
:FORWARD ACCEPT [0:0]
:OUTPUT ACCEPT [0:0]

# 允许loopback
-A INPUT -i lo -j ACCEPT

# 允许已建立的连接
-A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# 允许SSH
-A INPUT -p tcp --dport 22 -j ACCEPT

# 允许HTTPS
-A INPUT -p tcp --dport 443 -j ACCEPT

# 允许WireGuard
-A INPUT -p udp --dport 51820 -j ACCEPT

# 网络分段规则
-A FORWARD -i wg0 -o wg0 -j ACCEPT
-A FORWARD -i wg0 -o eth0 -j ACCEPT
-A FORWARD -i eth0 -o wg0 -m state --state ESTABLISHED,RELATED -j ACCEPT

# 默认拒绝
-A INPUT -j DROP
-A FORWARD -j DROP

COMMIT
EOF

# 应用规则
iptables-restore < /etc/iptables/rules.v4
```

### 数据保护

#### 1. 数据加密
```bash
# 配置数据库加密
cat > /etc/postgresql/15/main/conf.d/encryption.conf << 'EOF'
# 透明数据加密
ssl = on
ssl_cert_file = '/etc/ssl/certs/postgresql.crt'
ssl_key_file = '/etc/ssl/private/postgresql.key'

# 密码加密
password_encryption = scram-sha-256
EOF
```

#### 2. 备份加密
```bash
# 创建加密备份脚本
cat > /usr/local/bin/encrypted-backup.sh << 'EOF'
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/var/backups/wg-sdwan"
GPG_RECIPIENT="admin@example.com"

# 创建数据库备份
pg_dump -h localhost -U wg_user -d wg_sdwan | gzip | gpg --trust-model always --encrypt -r "$GPG_RECIPIENT" > "$BACKUP_DIR/db_backup_$DATE.sql.gz.gpg"

# 创建配置备份
tar -czf - /etc/wg-sdwan | gpg --trust-model always --encrypt -r "$GPG_RECIPIENT" > "$BACKUP_DIR/config_backup_$DATE.tar.gz.gpg"

echo "Encrypted backup completed"
EOF

chmod +x /usr/local/bin/encrypted-backup.sh
```

### 安全监控

#### 1. 入侵检测
```bash
# 安装和配置OSSEC
wget https://github.com/ossec/ossec-hids/archive/3.7.0.tar.gz
tar -xzf 3.7.0.tar.gz
cd ossec-hids-3.7.0
./install.sh

# 配置OSSEC
cat > /var/ossec/etc/ossec.conf << 'EOF'
<ossec_config>
  <global>
    <email_notification>yes</email_notification>
    <email_to>admin@example.com</email_to>
    <smtp_server>localhost</smtp_server>
    <email_from>ossec@example.com</email_from>
  </global>

  <syscheck>
    <directories check_all="yes">/etc/wg-sdwan</directories>
    <directories check_all="yes">/etc/wireguard</directories>
    <directories check_all="yes">/usr/local/bin</directories>
  </syscheck>

  <localfile>
    <log_format>syslog</log_format>
    <location>/var/log/wg-sdwan/controller.log</location>
  </localfile>

  <localfile>
    <log_format>syslog</log_format>
    <location>/var/log/auth.log</location>
  </localfile>
</ossec_config>
EOF

# 启动OSSEC
/var/ossec/bin/ossec-control start
```

#### 2. 日志监控
```bash
# 配置日志监控规则
cat > /etc/logwatch/conf/services/wg-sdwan.conf << 'EOF'
Title = "WireGuard SD-WAN"
LogFile = wg-sdwan

*OnlyService = wg-sdwan
*RemoveHeaders
EOF

cat > /etc/logwatch/scripts/services/wg-sdwan << 'EOF'
#!/usr/bin/perl
use strict;
use warnings;

my $Debug = $ENV{'LOGWATCH_DEBUG'} || 0;
my $Detail = $ENV{'LOGWATCH_DETAIL_LEVEL'} || 0;

my %LoginFailures = ();
my %SecurityEvents = ();

while (defined(my $ThisLine = <STDIN>)) {
    chomp($ThisLine);
    
    if ($ThisLine =~ /login failed.*from (.+)/) {
        $LoginFailures{$1}++;
    } elsif ($ThisLine =~ /security event: (.+)/) {
        $SecurityEvents{$1}++;
    }
}

if (keys %LoginFailures) {
    print "\nLogin Failures:\n";
    foreach my $ip (sort keys %LoginFailures) {
        print "   $ip: $LoginFailures{$ip} attempts\n";
    }
}

if (keys %SecurityEvents) {
    print "\nSecurity Events:\n";
    foreach my $event (sort keys %SecurityEvents) {
        print "   $event: $SecurityEvents{$event} times\n";
    }
}
EOF

chmod +x /etc/logwatch/scripts/services/wg-sdwan
```

---

## ❓ 常见问题

### 安装问题

#### Q1: 如何解决Go版本不兼容问题？
```bash
# 卸载旧版本
sudo rm -rf /usr/local/go

# 下载新版本
wget https://golang.org/dl/go1.21.linux-amd64.tar.gz

# 安装
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz

# 设置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
```

#### Q2: PostgreSQL连接失败怎么办？
```bash
# 检查PostgreSQL状态
systemctl status postgresql

# 检查连接配置
sudo -u postgres psql -c "\l"

# 测试连接
psql -h localhost -U wg_user -d wg_sdwan -c "SELECT 1;"

# 重置密码
sudo -u postgres psql -c "ALTER USER wg_user WITH PASSWORD 'new_password';"
```

### 配置问题

#### Q3: WireGuard配置文件格式错误？
```bash
# 验证配置文件
wg-quick up wg0 --dry-run

# 检查语法
wg showconf wg0

# 重新生成配置
wg genkey | tee private.key | wg pubkey > public.key
```

#### Q4: 如何修改默认端口？
```bash
# 编辑配置文件
nano /etc/wg-sdwan/controller.yaml

# 修改端口设置
server:
  port: 8081

# 重启服务
systemctl restart wg-sdwan-controller

# 更新防火墙规则
ufw allow 8081/tcp
```

### 网络问题

#### Q5: 节点无法连接到Hub？
```bash
# 检查网络连通性
ping hub.example.com

# 检查端口
nc -zv hub.example.com 51820

# 检查防火墙
iptables -L -n | grep 51820
ufw status | grep 51820

# 检查NAT配置
iptables -t nat -L -n
```

#### Q6: 网络性能差怎么优化？
```bash
# 调整MTU大小
ip link set mtu 1420 dev wg0

# 优化内核参数
echo 'net.core.rmem_max=134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_max=134217728' >> /etc/sysctl.conf
sysctl -p

# 启用BBR拥塞控制
echo 'net.ipv4.tcp_congestion_control=bbr' >> /etc/sysctl.conf
sysctl -p
```

### 安全问题

#### Q7: 如何重置管理员密码？
```bash
# 使用命令行重置
./controller/controller --reset-admin-password \
  --username=admin \
  --password=NewSecurePassword123!

# 或通过数据库直接修改
sudo -u postgres psql -d wg_sdwan -c "UPDATE users SET password = crypt('NewPassword', gen_salt('bf')) WHERE username = 'admin';"
```

#### Q8: JWT令牌过期怎么处理？
```bash
# 刷新令牌
curl -X POST https://wg-sdwan.example.com/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "YOUR_REFRESH_TOKEN"}'

# 重新登录
curl -X POST https://wg-sdwan.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password"}'
```

### 监控问题

#### Q9: Prometheus指标无法获取？
```bash
# 检查指标端点
curl http://localhost:9090/metrics

# 检查防火墙
ufw allow 9090/tcp

# 检查配置
grep -A5 -B5 "metrics" /etc/wg-sdwan/controller.yaml
```

#### Q10: 如何查看详细日志？
```bash
# 查看服务日志
journalctl -u wg-sdwan-controller -f

# 查看应用日志
tail -f /var/log/wg-sdwan/controller.log

# 增加日志级别
# 在配置文件中设置
logging:
  level: debug
```

### 性能问题

#### Q11: 系统响应慢怎么优化？
```bash
# 检查系统资源
top
htop
iostat -x 1

# 优化数据库
sudo -u postgres psql -d wg_sdwan -c "VACUUM ANALYZE;"

# 调整连接池
# 在配置文件中设置
database:
  max_connections: 200
  max_idle_connections: 50
```

#### Q12: 内存使用过高怎么处理？
```bash
# 查看内存使用
free -h
ps aux --sort=-%mem | head

# 调整应用内存限制
# 在systemd服务文件中添加
[Service]
MemoryMax=2G
MemoryHigh=1.5G

# 重启服务
systemctl daemon-reload
systemctl restart wg-sdwan-controller
```

---

## 📞 技术支持

### 获取帮助

1. **官方文档**: https://docs.wg-hubspoke.com
2. **GitHub Issues**: https://github.com/wg-hubspoke/wg-hubspoke/issues
3. **社区论坛**: https://community.wg-hubspoke.com
4. **技术博客**: https://blog.wg-hubspoke.com

### 报告问题

请在报告问题时提供以下信息：
- 系统版本和配置
- 错误日志和堆栈跟踪
- 重现步骤
- 网络拓扑图
- 诊断脚本输出

### 贡献代码

欢迎贡献代码和改进建议：
1. Fork项目仓库
2. 创建功能分支
3. 提交代码变更
4. 发起Pull Request
5. 等待代码审查

---

## 📄 许可证

本项目采用MIT许可证，详情请参阅LICENSE文件。

---

**版本**: 1.0.0  
**更新日期**: 2024年1月15日  
**维护者**: WireGuard SD-WAN Team