module github.com/wg-hubspoke/wg-hubspoke/agent

go 1.21

replace github.com/wg-hubspoke/wg-hubspoke/common => ../common

require (
	github.com/google/uuid v1.3.0
	github.com/spf13/cobra v1.7.0
	github.com/spf13/viper v1.16.0
	github.com/wg-hubspoke/wg-hubspoke/common v0.0.0-00010101000000-000000000000
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20230429144221-925a1e7659e6
	gopkg.in/yaml.v3 v3.0.1
)