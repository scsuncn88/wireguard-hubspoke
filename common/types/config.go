package types

import "time"

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Auth     AuthConfig     `yaml:"auth"`
	WG       WGConfig       `yaml:"wireguard"`
	Log      LogConfig      `yaml:"log"`
	JWT      JWTConfig      `yaml:"jwt"`
	HA       HAConfig       `yaml:"ha"`
}

type ServerConfig struct {
	Host         string        `yaml:"host" env:"CONTROLLER_HOST"`
	Port         int           `yaml:"port" env:"CONTROLLER_PORT"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	TLS          TLSConfig     `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" env:"TLS_ENABLED"`
	CertFile string `yaml:"cert_file" env:"TLS_CERT_FILE"`
	KeyFile  string `yaml:"key_file" env:"TLS_KEY_FILE"`
}

type DatabaseConfig struct {
	Host         string `yaml:"host" env:"DB_HOST"`
	Port         int    `yaml:"port" env:"DB_PORT"`
	Name         string `yaml:"name" env:"DB_NAME"`
	User         string `yaml:"user" env:"DB_USER"`
	Password     string `yaml:"password" env:"DB_PASSWORD"`
	SSLMode      string `yaml:"ssl_mode" env:"DB_SSL_MODE"`
	MaxConns     int    `yaml:"max_conns" env:"DB_MAX_CONNECTIONS"`
	MaxIdleTime  time.Duration `yaml:"max_idle_time" env:"DB_MAX_IDLE_TIME"`
}

type RedisConfig struct {
	Host     string `yaml:"host" env:"REDIS_HOST"`
	Port     int    `yaml:"port" env:"REDIS_PORT"`
	Password string `yaml:"password" env:"REDIS_PASSWORD"`
	DB       int    `yaml:"db" env:"REDIS_DB"`
}

type AuthConfig struct {
	JWTSecret    string        `yaml:"jwt_secret" env:"JWT_SECRET"`
	JWTExpiration time.Duration `yaml:"jwt_expiration" env:"JWT_EXPIRATION"`
	BCryptCost   int           `yaml:"bcrypt_cost" env:"BCRYPT_COST"`
}

type WGConfig struct {
	Interface        string `yaml:"interface" env:"WG_INTERFACE"`
	Subnet           string `yaml:"subnet" env:"WG_SUBNET"`
	PortRangeStart   int    `yaml:"port_range_start" env:"WG_PORT_RANGE_START"`
	PortRangeEnd     int    `yaml:"port_range_end" env:"WG_PORT_RANGE_END"`
	PersistentKeepalive int `yaml:"persistent_keepalive" env:"WG_PERSISTENT_KEEPALIVE"`
	MTU              int    `yaml:"mtu" env:"WG_MTU"`
	ConfigPath       string `yaml:"config_path" env:"WG_CONFIG_PATH"`
}

type LogConfig struct {
	Level  string `yaml:"level" env:"LOG_LEVEL"`
	Format string `yaml:"format" env:"LOG_FORMAT"`
	File   string `yaml:"file" env:"LOG_FILE"`
}

type JWTConfig struct {
	Secret    string        `yaml:"secret" env:"JWT_SECRET"`
	ExpiresIn time.Duration `yaml:"expires_in" env:"JWT_EXPIRES_IN"`
}

type HAConfig struct {
	Enabled           bool          `yaml:"enabled" env:"HA_ENABLED"`
	NodeID            string        `yaml:"node_id" env:"HA_NODE_ID"`
	ClusterID         string        `yaml:"cluster_id" env:"HA_CLUSTER_ID"`
	PeerNodes         []string      `yaml:"peer_nodes" env:"HA_PEER_NODES"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" env:"HA_HEARTBEAT_INTERVAL"`
	ElectionTimeout   time.Duration `yaml:"election_timeout" env:"HA_ELECTION_TIMEOUT"`
}