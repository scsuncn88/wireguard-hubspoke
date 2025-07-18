module github.com/wg-hubspoke/wg-hubspoke/controller

go 1.21

replace github.com/wg-hubspoke/wg-hubspoke/common => ../common

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/google/uuid v1.3.0
	github.com/lib/pq v1.10.9
	github.com/prometheus/client_golang v1.16.0
	github.com/redis/go-redis/v9 v9.0.5
	github.com/swaggo/gin-swagger v1.6.0
	github.com/swaggo/swag v1.16.1
	github.com/wg-hubspoke/wg-hubspoke/common v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.11.0
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20230429144221-925a1e7659e6
	gorm.io/driver/postgres v1.5.2
	gorm.io/gorm v1.25.2
)