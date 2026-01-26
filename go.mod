module github.com/launix-de/memcp

go 1.22.0

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
	github.com/launix-de/NonLockingReadMap v1.0.8
	github.com/launix-de/go-mysqlstack v0.0.0-20241101205441-bc39b4e0fb04
	github.com/launix-de/go-packrat/v2 v2.1.15
	github.com/ulikunitz/xz v0.5.15
	golang.org/x/text v0.21.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c // indirect
	golang.org/x/sys v0.26.0 // indirect
)

replace github.com/launix-de/NonLockingReadMap => ./third_party/NonLockingReadMap
