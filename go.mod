module github.com/launix-de/memcp

go 1.24.0

toolchain go1.24.12

require (
	github.com/chzyer/readline v1.5.1
	github.com/dc0d/onexit v1.1.0
	github.com/docker/go-units v0.5.0
	github.com/fsnotify/fsnotify v1.8.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/google/btree v1.1.3
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/jtolds/gls v4.20.0+incompatible
	github.com/launix-de/NonLockingReadMap v1.0.9
	github.com/launix-de/go-mysqlstack v0.0.0-20241101205441-bc39b4e0fb04
	github.com/launix-de/go-packrat/v2 v2.1.15
	github.com/ulikunitz/xz v0.5.15
	golang.org/x/text v0.21.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.32.7 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.7 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.96.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.6 // indirect
	github.com/aws/smithy-go v1.24.0 // indirect
	github.com/ceph/go-ceph v0.37.0 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c // indirect
	golang.org/x/sys v0.38.0 // indirect
)

replace github.com/launix-de/NonLockingReadMap => ./third_party/NonLockingReadMap
