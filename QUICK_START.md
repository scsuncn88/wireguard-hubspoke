# WireGuard SD-WAN 快速开始指南

## 🚀 5分钟快速部署

### 前提条件
- Linux服务器 (Ubuntu 20.04+)
- 2GB+ 内存
- 公网IP地址
- root权限

### 步骤1：一键安装
```bash
# 下载安装脚本
curl -fsSL https://raw.githubusercontent.com/wg-hubspoke/wg-hubspoke/main/install.sh | sudo bash

# 或手动安装
wget https://github.com/wg-hubspoke/wg-hubspoke/releases/latest/download/wg-hubspoke-linux-amd64.tar.gz
tar -xzf wg-hubspoke-linux-amd64.tar.gz
sudo ./install.sh
```

### 步骤2：配置环境
```bash
# 复制配置模板
sudo cp /etc/wg-sdwan/controller.yaml.example /etc/wg-sdwan/controller.yaml

# 编辑配置（修改数据库密码等）
sudo nano /etc/wg-sdwan/controller.yaml

# 初始化数据库
sudo wg-sdwan-controller --migrate
```

### 步骤3：启动服务
```bash
# 启动控制器
sudo systemctl start wg-sdwan-controller
sudo systemctl enable wg-sdwan-controller

# 检查状态
sudo systemctl status wg-sdwan-controller
```

### 步骤4：创建管理员
```bash
# 创建管理员账户
sudo wg-sdwan-controller --create-admin \
  --username=admin \
  --email=admin@example.com \
  --password=SecurePassword123!
```

### 步骤5：访问Web界面
1. 打开浏览器访问: `https://YOUR_SERVER_IP:8080`
2. 使用管理员账户登录
3. 开始配置您的SD-WAN网络

## 🔧 基本配置

### 添加Hub节点
1. 登录Web界面
2. 导航到"节点管理"
3. 点击"添加节点"
4. 选择"Hub"类型
5. 填写节点信息
6. 保存配置

### 添加Spoke节点
1. 在"节点管理"中点击"添加节点"
2. 选择"Spoke"类型
3. 填写节点信息
4. 下载生成的配置文件
5. 在目标服务器上应用配置

### 配置示例
```yaml
# Hub节点配置
name: "hub-main"
node_type: "hub"
endpoint: "hub.example.com:51820"
subnet: "10.100.0.0/16"

# Spoke节点配置
name: "spoke-branch1"
node_type: "spoke"
hub_endpoint: "hub.example.com:51820"
```

## 📚 更多资源

- [完整部署指南](./DEPLOYMENT_GUIDE.md)
- [API文档](./docs/api/)
- [故障排除](./docs/troubleshooting/)
- [最佳实践](./docs/best-practices/)

## 💡 获取帮助

- 📧 邮箱: support@wg-hubspoke.com
- 💬 社区: https://community.wg-hubspoke.com
- 🐛 问题报告: https://github.com/wg-hubspoke/issues

---

**需要高级功能？** 查看[完整部署指南](./DEPLOYMENT_GUIDE.md)了解企业级部署、监控、安全配置等详细信息。