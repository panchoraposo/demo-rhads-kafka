// Intentionally pins older community modules so RHDA / Syft / ACS show CVEs in the demo.
module ${{values.module_path}}

go 1.21

require (
	github.com/IBM/sarama v1.38.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/jackc/pgx/v5 v5.5.5
	github.com/prometheus/client_golang v1.16.0
	golang.org/x/net v0.17.0
	gopkg.in/yaml.v2 v2.4.0
)
