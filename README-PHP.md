<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# Embedded PHP

MemCP can optionally host PHP applications in its own process using the
versioned FrankenPHP Go library. The host is application-independent: choose
an application's document root, URL mount, thread count, and optional
front controller. WordPress is one example; other PHP applications can use the
same host and their usual PHP extensions.

No PHP interpreter, FrankenPHP implementation, WordPress, or other external
application sources are copied into this repository. Install those dependencies
outside the checkout. `phpbridge/` contains only MemCP's own PDO integration.

## Build

Use Go 1.26 or newer and a C toolchain. Install PHP 8.5 with ZTS and the embed
SAPI, including PDO and OPcache. The PHP headers, `php-config`, `libphp`, and
extensions must belong to the same build. A usual NTS PHP-FPM package is not
sufficient. Follow the upstream [FrankenPHP build instructions](https://frankenphp.dev/docs/compile/).

```sh
make php PHP_CONFIG=/path/to/php-zts/bin/php-config
```

This produces `memcp-php`. The executable dynamically links external `libphp`
from the selected installation; it does not start PHP-FPM or a separate
FrankenPHP/Caddy process. Install application-specific extensions such as
`mysqli`, `pdo_mysql`, `pdo_pgsql`, `pdo_sqlite`, `intl`, or `mbstring` in that
PHP installation. Its `php.ini` remains the place for PHP and OPcache settings.

`make` still produces the ordinary `memcp` executable without PHP or native
PHP build requirements. The shared Go module graph now requires Go 1.26 due
to FrankenPHP. The PHP build disables optional Watcher, Brotli and Mercure
features; no Caddy dependency is added.

## Mount applications from Scheme

`servePHP` returns a `(req res)` handler, just like `serveStatic`. It creates no
listener. Mount any number of applications in the existing Scheme HTTP router;
all mounts share the process-wide FrankenPHP thread pool.

For example, place this in a Scheme module loaded after `lib/main.scm`:

```scheme
(define blog (servePHP "/srv/blog/public" "/blog" "index.php"))
(define wiki (servePHP "/srv/wiki" "/wiki" "index.php"))
(define http_handler (begin
    (define previous http_handler)
    (lambda (req res)
        (match (req "path")
            (regex "^/blog(/|$)" _ _) (blog req res)
            (regex "^/wiki(/|$)" _ _) (wiki req res)
            _ (previous req res)))))
```

```sh
./memcp-php --no-repl -data /path/to/persistent/memcp-data \
  --api-port=8080 --php-threads=4 lib/main.scm /path/to/apps.scm
```

The arguments are document root, optional URL prefix, and optional fallback PHP
filename. Relative roots resolve against the importing Scheme file. An empty
prefix mounts at `/`; without a fallback, missing files return 404. For an
independent listener, explicitly call `(serve port app_handler)` from Scheme,
using the same HTTP machinery as every other handler. Its optional third
argument binds an address, e.g. `(serve 8080 app_handler "127.0.0.1")`.
The former
`--php-root`, `--php-listen` and `--php-front-controller` options are removed.

PHP receives the original request URI, query, body, cookies and headers;
`SCRIPT_NAME` and `PHP_SELF` include the mount prefix, while script lookup stays
inside the configured document root. Existing directories use `index.php` or
`index.html`, and PHP `PATH_INFO` is supported. Static assets remain within the
root; dotfiles, PHP source backups and escaping symlinks are rejected. Routing
and HTTP authentication can run in Scheme before invoking the PHP handler.

The shared PHP runtime starts lazily after Scheme bootstrap, on its first
request. `--php-threads` configures its fixed thread count. SIGTERM/SIGINT drains
the Scheme HTTP servers before shutting down PHP and storage.

The host uses classic PHP request lifecycles. Request globals and ordinary
objects are released after each request; OPcache stays warm. Long-lived PHP
application workers are not enabled by this integration. Native extension
crashes affect the entire process, including MemCP.

## PDO connections

The addon registers **only `memcp:`**. It neither replaces `PDO` nor intercepts
other DSNs. Existing driver extensions continue to resolve their own schemes:

```php
$local = new PDO('memcp:dbname=shop', $user, $password);
$mysql = new PDO('mysql:host=db.example;dbname=shop', $user, $password);
$sqlite = new PDO('sqlite:/path/to/cache.sqlite');
```

Create the MemCP database and account using the existing SQL administration
interface before connecting. Authentication uses MemCP's user catalog, and
queries use its existing SQL permission checks, session state, transactions
and planner cache. `memcp:` calls directly into the Go/Scheme SQL frontend:
there is no socket or MySQL wire encoding on this path. Other DSNs retain their
normal connection behavior and do not require a local MemCP database.

The initial driver supports queries, named/positional prepared statements,
`bindValue`/`bindParam`, forward fetching, affected rows, insert IDs, and
begin/commit/rollback. Prepared statements use PDO's emulation and byte-safe
hex literals. They do not yet provide native parameter binding or persistent
connections; unsupported modes fail explicitly. Values retain MemCP's runtime
numeric types (an integral SQL literal may be a PHP float).

### Result callbacks and sessions

Every `new PDO('memcp:...', user, password)` authenticates against `mysql_auth`
before a connection handle is created. An HTTP login or PHP session does not
bypass this check. Database existence is checked separately, and every query
uses the existing SQL permission checks for that authenticated username.

Each PDO connection owns a separate Scheme session initialized with `username`
and `schema`, plus a registered `SessionState`. Neither is borrowed from the
HTTP request's `__session` or from PHP's `$_SESSION`. Transactions, connection
variables and insert IDs therefore belong to the PDO connection. Persistent
PDO connections are not supported.

The Go bridge creates stable row/metadata callback closures when the connection
is opened and wraps them with `scm.NewFunc`. On each query it calls the existing
Scheme `mysql_handler` with `(schema, sql, resultrow_sql, resultfields_sql,
connection_session, session_state, query_sequence)`. SQL result execution calls
these supplied closures; no global result handler is replaced.

A connection mutex serializes query/close. The result collector has a separate
mutex because storage workers may invoke the row callback concurrently. The
synchronous SQL call completes its workers before the collector is copied and
reset. Reusable cell/byte buffers are each bounded to 64 KiB of retained capacity;
larger buffers are released after the query. Metadata keys are cleared between
queries. Rows are written directly into the collector's flat cell array.

Results are limited to 64 MiB and copied into independent C-owned memory in one
batch. Existing PDO statements retain their own data when another query reuses
the collector; fetching rows never calls back into Go. Strings preserve arbitrary
bytes. Use SQL pagination for larger results. The 30-second query timeout,
process-list tracking, cancellation and transaction handling remain active.
Closing a connection rolls back unfinished transactions, releases its locks and
unregisters its session, including during request teardown after PHP exceptions.

Applications or ORMs that recognize only specific PDO driver names may need a
MemCP/MySQL-dialect adapter. Applications using `mysqli`, including unmodified
WordPress, can instead use MemCP's existing MySQL TCP/Unix socket frontend.
That path uses the MySQL protocol even though PHP runs in the same process.
A native PDO driver alone cannot transparently replace `mysqli`.

## Verification

Install `pdo_sqlite` and `pdo_mysql` alongside PDO in the ZTS build, then run:

```sh
make test-php PHP_CONFIG=/path/to/php-zts/bin/php-config
```

`test-php` creates a disposable server and database. It checks ZTS/OPcache,
coexistence with MySQL wire and SQLite, authentication failures, binary
parameters, request teardown rollback, independent parallel requests, and HTTP
file boundaries. It also checks multiple Scheme mounts, POST/cookies/headers,
independent statement buffers and concurrent result callbacks. Full SQL tests
run in CI; the PHP job depends on the successful SQL job and a PHP-related path
guard.
It contains only our own PHP test fixtures and does not download applications.

For a WordPress demonstration, download WordPress and WP-CLI into a directory
outside the checkout. Configure a dedicated persistent MemCP database and
Unix socket in `wp-config.php`, mount the WordPress directory with `servePHP`
in a Scheme module, and run the normal WordPress installer. Keep credentials,
uploaded files, and the MemCP data directory outside the source tree.
