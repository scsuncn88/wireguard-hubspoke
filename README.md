# WireGuard SD-WAN

A comprehensive WireGuard-based Hub-and-Spoke SD-WAN management system with centralized control, automated configuration, and high availability.

## Features

- **Hub-and-Spoke Architecture**: Centralized hub nodes routing traffic between distributed spoke nodes
- **Automated Configuration**: Automatic WireGuard configuration generation and distribution
- **Web Management Interface**: Real-time topology visualization and policy management
- **High Availability**: Multi-hub support with failover mechanisms
- **RESTful API**: Comprehensive API for automation and integration
- **Access Control**: Fine-grained ACL and policy management
- **Real-time Monitoring**: Connection status, traffic statistics, and health monitoring
- **Containerized Deployment**: Docker and Kubernetes support

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Controller    │    │   Web UI        │    │   Agent         │
│   (Go/API)      │◄──►│   (React/TS)    │    │   (Go/Daemon)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                                              │
         ▼                                              ▼
┌─────────────────┐                            ┌─────────────────┐
│   Database      │                            │   WireGuard     │
│   (PostgreSQL)  │                            │   (Linux)       │
└─────────────────┘                            └─────────────────┘
```

## Project Structure

```
wg-hubspoke/
├── controller/          # Control plane services
│   ├── api/            # REST API handlers
│   ├── topology/       # Topology management
│   ├── ha/             # High availability logic
│   └── models/         # Database models
├── agent/              # Node-side daemon
│   ├── client/         # Registration client
│   ├── config/         # Configuration management
│   └── wg/             # WireGuard integration
├── ui/                 # Web UI frontend
│   ├── components/     # React components
│   ├── services/       # API services
│   └── layouts/        # Page layouts
├── cli/                # Command line tools
├── common/             # Shared utilities
├── infra/              # Deployment manifests
│   ├── docker/         # Docker configurations
│   ├── k8s/            # Kubernetes manifests
│   └── helm/           # Helm charts
├── tests/              # Test suites
└── docs/               # Documentation
```

## Quick Start

### Prerequisites

- Ubuntu 20.04+ or compatible Linux distribution
- Docker and Docker Compose
- WireGuard kernel module
- Go 1.21+ (for development)
- Node.js 18+ (for UI development)

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-org/wg-hubspoke.git
   cd wg-hubspoke
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Activate Python virtual environment**
   ```bash
   source venv_linux/bin/activate
   ```

4. **Start development environment**
   ```bash
   docker-compose -f docker-compose.dev.yml up -d
   ```

5. **Initialize database**
   ```bash
   make db-migrate
   ```

6. **Start the controller**
   ```bash
   cd controller
   go run main.go
   ```

7. **Start the web UI**
   ```bash
   cd ui
   npm install
   npm start
   ```

### Production Deployment

#### Using Docker Compose

```bash
# Copy and configure environment
cp .env.example .env.prod
# Edit .env.prod with production values

# Deploy with Docker Compose
docker-compose -f docker-compose.prod.yml up -d
```

#### Using Kubernetes

```bash
# Install with Helm
helm install wireguard-sdwan ./infra/helm/ \
  --set controller.image.tag=v1.0.0 \
  --set database.password=your_secure_password
```

## Configuration

### Environment Variables

Key configuration options in `.env`:

```bash
# Controller
CONTROLLER_HOST=0.0.0.0
CONTROLLER_PORT=8080
DB_HOST=localhost
DB_NAME=wireguard_sdwan

# WireGuard
WG_SUBNET=10.100.0.0/16
WG_PORT_RANGE_START=51820
WG_PORT_RANGE_END=51870

# Security
JWT_SECRET=your_jwt_secret_here
TLS_ENABLED=true
```

### WireGuard Setup

1. **Install WireGuard**
   ```bash
   sudo apt update
   sudo apt install wireguard
   ```

2. **Configure kernel parameters**
   ```bash
   echo 'net.ipv4.ip_forward=1' | sudo tee -a /etc/sysctl.conf
   sudo sysctl -p
   ```

3. **Set up firewall rules**
   ```bash
   sudo ufw allow 51820/udp
   sudo ufw allow 8080/tcp
   ```

## API Documentation

The API documentation is available at:
- Development: http://localhost:8080/docs
- Swagger UI: http://localhost:8080/swagger

### Key Endpoints

- `POST /api/v1/nodes` - Register new node
- `GET /api/v1/nodes` - List all nodes
- `GET /api/v1/topology` - Get network topology
- `POST /api/v1/policies` - Create access policy
- `GET /api/v1/health` - Health check

## Usage

### Register a Spoke Node

```bash
# Using the agent
sudo ./agent --controller-url=https://controller.example.com \
             --node-name=spoke-01 \
             --node-type=spoke

# Using the CLI
./cli node register --name=spoke-01 --type=spoke --endpoint=spoke-01.example.com
```

### Create Hub Node

```bash
./cli node register --name=hub-01 --type=hub --endpoint=hub.example.com:51820
```

### Manage Policies

```bash
# Allow spoke-01 to access hub-01
./cli policy create --source=spoke-01 --destination=hub-01 --action=allow

# Deny communication between spoke nodes
./cli policy create --source-group=spokes --destination-group=spokes --action=deny
```

## Monitoring

### Health Checks

```bash
# Check controller health
curl http://localhost:8080/health

# Check agent health
curl http://localhost:8081/health
```

### Metrics

Prometheus metrics are available at:
- Controller: http://localhost:8080/metrics
- Agent: http://localhost:8081/metrics

### Grafana Dashboard

Import the provided dashboard from `./monitoring/grafana/dashboard.json`

## Development

### Building

```bash
# Build all components
make build

# Build specific component
make build-controller
make build-agent
make build-ui
```

### Testing

```bash
# Run all tests
make test

# Run specific test suites
make test-unit
make test-integration
make test-e2e
```

### Code Quality

```bash
# Run linting
make lint

# Format code
make format

# Security scan
make security-scan
```

## Troubleshooting

### Common Issues

1. **WireGuard interface not found**
   ```bash
   sudo modprobe wireguard
   sudo systemctl restart wireguard-sdwan-agent
   ```

2. **Database connection failed**
   ```bash
   # Check database status
   docker-compose ps database
   
   # Check logs
   docker-compose logs database
   ```

3. **Agent registration failed**
   ```bash
   # Check controller logs
   journalctl -u wireguard-sdwan-controller
   
   # Check network connectivity
   curl -v http://controller.example.com:8080/health
   ```

### Debugging

Enable debug logging:
```bash
export LOG_LEVEL=debug
export DEBUG_MODE=true
```

View detailed logs:
```bash
# Controller logs
tail -f /var/log/wireguard-sdwan/controller.log

# Agent logs
journalctl -u wireguard-sdwan-agent -f
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Update documentation
6. Submit a pull request

### Development Guidelines

- Follow Go standard formatting with `gofmt`
- Write unit tests for new functionality
- Update API documentation
- Follow security best practices
- Test with multiple Linux distributions

## Security

- Private keys never leave the local node
- All control plane communication uses TLS
- JWT tokens for API authentication
- Input validation and sanitization
- Regular security audits

## Performance

- Supports 1000+ concurrent nodes
- Sub-second configuration distribution
- Optimized database queries
- Efficient WireGuard configuration generation

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- Documentation: [docs/](docs/)
- Issues: [GitHub Issues](https://github.com/your-org/wg-hubspoke/issues)
- Discussions: [GitHub Discussions](https://github.com/your-org/wg-hubspoke/discussions)

## Acknowledgments

- [WireGuard](https://www.wireguard.com/) - Modern VPN protocol
- [Netmaker](https://github.com/gravitl/netmaker) - SD-WAN reference architecture
- [wg-meshconf](https://github.com/k4yt3x/wg-meshconf) - Configuration generation inspiration