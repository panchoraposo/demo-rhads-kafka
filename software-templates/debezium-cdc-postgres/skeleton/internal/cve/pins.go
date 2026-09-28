package cve

// Intentionally import vulnerable community packages so RHDA/Syft/ACS surface CVEs.
import (
	_ "github.com/dgrijalva/jwt-go"
	_ "github.com/gorilla/websocket"
	_ "github.com/prometheus/client_golang/prometheus"
	_ "golang.org/x/net/html"
	_ "gopkg.in/yaml.v2"
)
