# WireGuard SD-WAN API 参考文档

## 📋 目录
1. [API概览](#api概览)
2. [认证授权](#认证授权)
3. [节点管理](#节点管理)
4. [用户管理](#用户管理)
5. [监控接口](#监控接口)
6. [配置管理](#配置管理)
7. [审计日志](#审计日志)
8. [系统接口](#系统接口)
9. [错误处理](#错误处理)
10. [SDK示例](#sdk示例)

---

## 🌐 API概览

### 基础信息
- **Base URL**: `https://wg-sdwan.example.com/api/v1`
- **协议**: HTTPS
- **格式**: JSON
- **认证**: JWT Bearer Token
- **版本**: v1

### 通用响应格式
```json
{
  "success": true,
  "data": {},
  "error": "",
  "message": "",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 分页响应格式
```json
{
  "success": true,
  "data": [],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

---

## 🔐 认证授权

### 用户登录
```http
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "password123"
}
```

**响应**:
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

### 用户注册
```http
POST /auth/register
Content-Type: application/json

{
  "username": "newuser",
  "email": "newuser@example.com",
  "password": "SecurePassword123!",
  "role": "user"
}
```

### 刷新令牌
```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### 修改密码
```http
POST /auth/change-password
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "current_password": "oldpassword",
  "new_password": "newpassword123"
}
```

### 用户退出
```http
POST /auth/logout
Authorization: Bearer YOUR_TOKEN
```

---

## 🖥️ 节点管理

### 获取节点列表
```http
GET /nodes?page=1&per_page=20&node_type=hub&status=active
Authorization: Bearer YOUR_TOKEN
```

**查询参数**:
- `page`: 页码（默认1）
- `per_page`: 每页数量（默认20）
- `node_type`: 节点类型（hub/spoke）
- `status`: 状态（active/inactive/pending）
- `search`: 搜索关键词

**响应**:
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "hub-main",
      "node_type": "hub",
      "public_key": "ABC123DEF456...",
      "allocated_ip": "10.100.1.1",
      "endpoint": "hub.example.com:51820",
      "status": "active",
      "description": "Main hub node",
      "last_seen": "2024-01-15T10:30:00Z",
      "created_at": "2024-01-15T09:00:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
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

### 创建节点
```http
POST /nodes
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "name": "hub-backup",
  "node_type": "hub",
  "public_key": "XYZ789ABC123...",
  "endpoint": "hub-backup.example.com:51820",
  "description": "Backup hub node"
}
```

### 获取单个节点
```http
GET /nodes/{node_id}
Authorization: Bearer YOUR_TOKEN
```

### 更新节点
```http
PUT /nodes/{node_id}
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "name": "updated-hub-name",
  "description": "Updated description",
  "endpoint": "new-endpoint.example.com:51820"
}
```

### 删除节点
```http
DELETE /nodes/{node_id}
Authorization: Bearer YOUR_TOKEN
```

### 获取节点配置
```http
GET /nodes/{node_id}/config
Authorization: Bearer YOUR_TOKEN
```

**响应**:
```json
{
  "success": true,
  "data": {
    "config": "[Interface]\nPrivateKey = [PRIVATE_KEY]\nAddress = 10.100.1.1/24\nListenPort = 51820\n\n[Peer]\nPublicKey = ABC123DEF456...\nAllowedIPs = 10.100.0.0/16\nEndpoint = hub.example.com:51820\nPersistentKeepalive = 25",
    "qr_code": "data:image/png;base64,iVBORw0KGgoAAAANS..."
  }
}
```

### 节点状态控制
```http
POST /nodes/{node_id}/status
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "status": "active"
}
```

---

## 👥 用户管理

### 获取用户列表
```http
GET /users?page=1&per_page=20&role=admin&active=true
Authorization: Bearer YOUR_TOKEN
```

**查询参数**:
- `page`: 页码
- `per_page`: 每页数量
- `role`: 用户角色（admin/user）
- `active`: 是否活跃（true/false）
- `search`: 搜索关键词

### 创建用户
```http
POST /users
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "username": "operator",
  "email": "operator@example.com",
  "password": "SecurePassword123!",
  "role": "user",
  "active": true
}
```

### 获取单个用户
```http
GET /users/{user_id}
Authorization: Bearer YOUR_TOKEN
```

### 更新用户
```http
PUT /users/{user_id}
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "email": "newemail@example.com",
  "role": "admin",
  "active": false
}
```

### 删除用户
```http
DELETE /users/{user_id}
Authorization: Bearer YOUR_TOKEN
```

### 重置用户密码
```http
POST /users/{user_id}/reset-password
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "new_password": "NewSecurePassword123!"
}
```

---

## 📊 监控接口

### 更新节点指标
```http
POST /monitoring/nodes/{node_id}/metrics
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "cpu_usage": 25.5,
  "memory_usage": 60.2,
  "disk_usage": 45.8,
  "network_rx": 1024000,
  "network_tx": 512000,
  "wireguard_peers": 5,
  "active_connections": 12,
  "uptime": 3600,
  "load_average": [0.5, 0.6, 0.7]
}
```

### 获取节点指标
```http
GET /monitoring/nodes/{node_id}/metrics?period=1h
Authorization: Bearer YOUR_TOKEN
```

**查询参数**:
- `period`: 时间段（5m/1h/1d/1w/1M）
- `metrics`: 指标类型（cpu/memory/disk/network）

**响应**:
```json
{
  "success": true,
  "data": {
    "node_id": "550e8400-e29b-41d4-a716-446655440000",
    "metrics": {
      "cpu_usage": 25.5,
      "memory_usage": 60.2,
      "disk_usage": 45.8,
      "network_rx": 1024000,
      "network_tx": 512000,
      "wireguard_peers": 5,
      "active_connections": 12
    },
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### 获取所有节点指标
```http
GET /monitoring/nodes/metrics?period=1h
Authorization: Bearer YOUR_TOKEN
```

### 获取系统统计
```http
GET /monitoring/stats
Authorization: Bearer YOUR_TOKEN
```

**响应**:
```json
{
  "success": true,
  "data": {
    "total_nodes": 10,
    "active_nodes": 8,
    "hub_nodes": 2,
    "spoke_nodes": 8,
    "total_traffic": 1073741824,
    "active_connections": 45,
    "system_uptime": 86400
  }
}
```

### 获取网络拓扑
```http
GET /monitoring/topology
Authorization: Bearer YOUR_TOKEN
```

---

## ⚙️ 配置管理

### 导出配置
```http
GET /config/export?format=json&include=nodes,users,settings
Authorization: Bearer YOUR_TOKEN
```

**查询参数**:
- `format`: 格式（json/yaml/toml）
- `include`: 包含的配置项（nodes/users/settings）

**响应**:
```json
{
  "success": true,
  "data": {
    "version": "1.0.0",
    "exported_at": "2024-01-15T10:30:00Z",
    "nodes": [...],
    "users": [...],
    "settings": {...}
  }
}
```

### 导入配置
```http
POST /config/import
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "version": "1.0.0",
  "nodes": [...],
  "users": [...],
  "settings": {...}
}
```

### 获取系统配置
```http
GET /config/settings
Authorization: Bearer YOUR_TOKEN
```

### 更新系统配置
```http
PUT /config/settings
Authorization: Bearer YOUR_TOKEN
Content-Type: application/json

{
  "wireguard": {
    "subnet": "10.100.0.0/16",
    "port": 51820,
    "dns_servers": ["8.8.8.8", "8.8.4.4"]
  },
  "security": {
    "max_login_attempts": 5,
    "session_timeout": 3600
  }
}
```

---

## 📋 审计日志

### 获取审计日志
```http
GET /audit/logs?page=1&per_page=50&action=CREATE_NODE&user_id=123
Authorization: Bearer YOUR_TOKEN
```

**查询参数**:
- `page`: 页码
- `per_page`: 每页数量
- `action`: 操作类型
- `user_id`: 用户ID
- `resource`: 资源类型
- `start_date`: 开始日期
- `end_date`: 结束日期

**响应**:
```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "550e8400-e29b-41d4-a716-446655440001",
      "username": "admin",
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

### 获取单个审计日志
```http
GET /audit/logs/{log_id}
Authorization: Bearer YOUR_TOKEN
```

### 获取用户活动
```http
GET /audit/users/{user_id}/activity
Authorization: Bearer YOUR_TOKEN
```

### 获取资源操作历史
```http
GET /audit/resources/{resource_type}/{resource_id}/history
Authorization: Bearer YOUR_TOKEN
```

---

## 🔧 系统接口

### 健康检查
```http
GET /health
```

**响应**:
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

### 就绪检查
```http
GET /ready
```

### 系统信息
```http
GET /info
Authorization: Bearer YOUR_TOKEN
```

**响应**:
```json
{
  "success": true,
  "data": {
    "version": "1.0.0",
    "build_time": "2024-01-15T08:00:00Z",
    "git_commit": "abc123def456",
    "go_version": "go1.21.0",
    "os": "linux",
    "arch": "amd64"
  }
}
```

### Prometheus指标
```http
GET /metrics
```

### 系统状态
```http
GET /status
Authorization: Bearer YOUR_TOKEN
```

---

## ❌ 错误处理

### 错误响应格式
```json
{
  "success": false,
  "error": "VALIDATION_ERROR",
  "message": "Invalid input data",
  "details": {
    "field": "username",
    "code": "REQUIRED",
    "message": "Username is required"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 常见错误码

| HTTP状态码 | 错误码 | 描述 |
|-----------|--------|------|
| 400 | VALIDATION_ERROR | 输入验证错误 |
| 401 | UNAUTHORIZED | 未授权访问 |
| 403 | FORBIDDEN | 禁止访问 |
| 404 | NOT_FOUND | 资源不存在 |
| 409 | CONFLICT | 资源冲突 |
| 422 | UNPROCESSABLE_ENTITY | 无法处理的实体 |
| 429 | RATE_LIMIT_EXCEEDED | 超过速率限制 |
| 500 | INTERNAL_ERROR | 内部服务器错误 |

### 错误处理示例
```javascript
try {
  const response = await fetch('/api/v1/nodes', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(nodeData)
  });
  
  const result = await response.json();
  
  if (!result.success) {
    throw new Error(result.message || 'API request failed');
  }
  
  return result.data;
} catch (error) {
  console.error('API Error:', error);
  // 处理错误
}
```

---

## 💻 SDK示例

### JavaScript/TypeScript
```typescript
class WireGuardSDWANAPI {
  private baseURL: string;
  private token: string;

  constructor(baseURL: string, token: string) {
    this.baseURL = baseURL;
    this.token = token;
  }

  private async request(endpoint: string, options: RequestInit = {}): Promise<any> {
    const response = await fetch(`${this.baseURL}${endpoint}`, {
      ...options,
      headers: {
        'Authorization': `Bearer ${this.token}`,
        'Content-Type': 'application/json',
        ...options.headers
      }
    });

    const result = await response.json();
    
    if (!result.success) {
      throw new Error(result.message || 'API request failed');
    }
    
    return result.data;
  }

  async getNodes(filters: any = {}): Promise<any> {
    const params = new URLSearchParams(filters);
    return await this.request(`/nodes?${params}`);
  }

  async createNode(nodeData: any): Promise<any> {
    return await this.request('/nodes', {
      method: 'POST',
      body: JSON.stringify(nodeData)
    });
  }

  async getNodeConfig(nodeId: string): Promise<any> {
    return await this.request(`/nodes/${nodeId}/config`);
  }
}

// 使用示例
const api = new WireGuardSDWANAPI('https://wg-sdwan.example.com/api/v1', 'your-token');

// 获取节点列表
const nodes = await api.getNodes({ node_type: 'hub' });

// 创建节点
const newNode = await api.createNode({
  name: 'hub-main',
  node_type: 'hub',
  public_key: 'ABC123...',
  endpoint: 'hub.example.com:51820'
});
```

### Python
```python
import requests
import json
from typing import Dict, Any, Optional

class WireGuardSDWANAPI:
    def __init__(self, base_url: str, token: str):
        self.base_url = base_url
        self.token = token
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {token}',
            'Content-Type': 'application/json'
        })

    def _request(self, endpoint: str, method: str = 'GET', data: Optional[Dict] = None) -> Any:
        url = f"{self.base_url}{endpoint}"
        
        response = self.session.request(
            method=method,
            url=url,
            json=data if data else None
        )
        
        result = response.json()
        
        if not result.get('success', False):
            raise Exception(result.get('message', 'API request failed'))
        
        return result.get('data')

    def get_nodes(self, filters: Optional[Dict] = None) -> Dict:
        params = '&'.join([f"{k}={v}" for k, v in (filters or {}).items()])
        endpoint = f"/nodes?{params}" if params else "/nodes"
        return self._request(endpoint)

    def create_node(self, node_data: Dict) -> Dict:
        return self._request('/nodes', method='POST', data=node_data)

    def get_node_config(self, node_id: str) -> Dict:
        return self._request(f'/nodes/{node_id}/config')

    def update_node_metrics(self, node_id: str, metrics: Dict) -> Dict:
        return self._request(f'/monitoring/nodes/{node_id}/metrics', method='POST', data=metrics)

# 使用示例
api = WireGuardSDWANAPI('https://wg-sdwan.example.com/api/v1', 'your-token')

# 获取节点列表
nodes = api.get_nodes({'node_type': 'hub'})

# 创建节点
new_node = api.create_node({
    'name': 'hub-main',
    'node_type': 'hub',
    'public_key': 'ABC123...',
    'endpoint': 'hub.example.com:51820'
})
```

### Go
```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type WireGuardSDWANAPI struct {
    BaseURL string
    Token   string
    Client  *http.Client
}

func NewWireGuardSDWANAPI(baseURL, token string) *WireGuardSDWANAPI {
    return &WireGuardSDWANAPI{
        BaseURL: baseURL,
        Token:   token,
        Client:  &http.Client{},
    }
}

func (api *WireGuardSDWANAPI) request(endpoint, method string, data interface{}) (map[string]interface{}, error) {
    var reqBody []byte
    var err error
    
    if data != nil {
        reqBody, err = json.Marshal(data)
        if err != nil {
            return nil, err
        }
    }
    
    req, err := http.NewRequest(method, api.BaseURL+endpoint, bytes.NewBuffer(reqBody))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", "Bearer "+api.Token)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := api.Client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    if !result["success"].(bool) {
        return nil, fmt.Errorf("API error: %s", result["message"])
    }
    
    return result["data"].(map[string]interface{}), nil
}

func (api *WireGuardSDWANAPI) GetNodes(filters map[string]string) (map[string]interface{}, error) {
    endpoint := "/nodes"
    if len(filters) > 0 {
        endpoint += "?"
        for k, v := range filters {
            endpoint += fmt.Sprintf("%s=%s&", k, v)
        }
    }
    
    return api.request(endpoint, "GET", nil)
}

func (api *WireGuardSDWANAPI) CreateNode(nodeData map[string]interface{}) (map[string]interface{}, error) {
    return api.request("/nodes", "POST", nodeData)
}

// 使用示例
func main() {
    api := NewWireGuardSDWANAPI("https://wg-sdwan.example.com/api/v1", "your-token")
    
    // 获取节点列表
    nodes, err := api.GetNodes(map[string]string{"node_type": "hub"})
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Nodes: %+v\n", nodes)
}
```

### cURL示例
```bash
# 登录获取令牌
curl -X POST https://wg-sdwan.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "password123"}'

# 获取节点列表
curl -X GET https://wg-sdwan.example.com/api/v1/nodes \
  -H "Authorization: Bearer YOUR_TOKEN"

# 创建节点
curl -X POST https://wg-sdwan.example.com/api/v1/nodes \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "hub-main",
    "node_type": "hub",
    "public_key": "ABC123...",
    "endpoint": "hub.example.com:51820"
  }'

# 获取节点配置
curl -X GET https://wg-sdwan.example.com/api/v1/nodes/NODE_ID/config \
  -H "Authorization: Bearer YOUR_TOKEN"

# 更新节点指标
curl -X POST https://wg-sdwan.example.com/api/v1/monitoring/nodes/NODE_ID/metrics \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "cpu_usage": 25.5,
    "memory_usage": 60.2,
    "disk_usage": 45.8
  }'
```

---

## 📚 更多资源

- [完整部署指南](./DEPLOYMENT_GUIDE.md)
- [快速开始](./QUICK_START.md)
- [故障排除](./docs/troubleshooting.md)
- [开发者指南](./docs/developer-guide.md)

---

**版本**: 1.0.0  
**更新日期**: 2024年1月15日  
**维护者**: WireGuard SD-WAN Team