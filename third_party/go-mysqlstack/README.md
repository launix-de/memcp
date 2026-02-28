# go-mysqlstack

***go-mysqlstack*** is an MySQL protocol library implementing in Go (golang).

Protocol is based on [mysqlproto-go](https://github.com/pubnative/mysqlproto-go) and [go-sql-driver](https://github.com/go-sql-driver/mysql)

## Running Tests

```
$ mkdir src
$ export GOPATH=`pwd`
$ go get -u github.com/launix-de/go-mysqlstack/driver
$ cd src/github.com/launix-de/go-mysqlstack/
$ make test
```

## Examples

1. ***examples/mysqld.go*** mocks a MySQL server by running:

```
$ go run example/mysqld.go
  2018/01/26 16:02:02.304376 mysqld.go:52:     [INFO]    mysqld.server.start.address[:4407]
```

2. ***examples/client.go*** mocks a client and query from the mock MySQL server:

```
$ go run example/client.go
  2018/01/26 16:06:10.779340 client.go:32:    [INFO]    results:[[[10 nice name]]]
```

## Status

go-mysqlstack is production ready.

## License

go-mysqlstack is released under the BSD-3-Clause License. See [LICENSE](https://github.com/launix-de/go-mysqlstack/blob/master/LICENSE)
