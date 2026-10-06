// Intentionally pins older community modules so RHDA / Syft / ACS show CVEs in the demo.
module ${{values.module_path}}

go 1.21

require (
	github.com/IBM/sarama v1.43.3
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/gorilla/websocket v1.4.2
	github.com/jackc/pgx/v5 v5.5.5
	github.com/prometheus/client_golang v1.16.0
	golang.org/x/net v0.17.0
	gopkg.in/yaml.v2 v2.4.0
)

// Keep tidy on go-toolset 1.21 (newer go-internal needs go>=1.23).
replace github.com/rogpeppe/go-internal => github.com/rogpeppe/go-internal v1.12.0
