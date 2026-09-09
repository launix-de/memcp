<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# Embedded PHP

MemCP can optionally host PHP applications in its own process using the
versioned FrankenPHP Go library. The host is application-independent: choose
an application's document root, URL mount, thread count, and optional
front controller. WordPress is one example; other PHP applications can use the
same host and their usual PHP extensions.

No PHP interpreter, FrankenPHP implementation, WordPress, or other external
application sources are copied into this repository. Install those dependencies
outside the checkout. `phpbridge/` contains MemCP's own PDO, locale and isolated IMAP adapters.

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

For a single application, the standard Scheme bootstrap accepts a document root
relative to the current working directory (or an absolute path):

```sh
./memcp-php --serve /srv/my-app/public --api-port=8080 --mysql-port=3307
```

`--serve=PATH` is equivalent. `lib/main.scm` installs the PHP folder at `/`
with `index.php` as front controller, below the existing SQL/dashboard/RDF
handlers. `/dashboard`, its API/WebSocket routes, and the SQL API remain
available with their normal authentication. No application-specific Scheme
module is needed for this mode.

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
  --api-port=8080 lib/main.scm /path/to/apps.scm
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
request. `(settings "PHPThreads")` configures its fixed thread count. SIGTERM/SIGINT drains
the Scheme HTTP servers before shutting down PHP and storage.

The host uses classic PHP request lifecycles. Request globals and ordinary
objects are released after each request; OPcache stays warm. Long-lived PHP
application workers are not enabled by this integration. Native extension
crashes affect the entire process, including MemCP.

## Gettext and extension isolation

On Linux/glibc, `setlocale()` uses a request-owned thread locale. Gettext's
current domain, directory bindings and output encodings also belong to the
request. Applications can keep their existing `putenv`, `setlocale`,
`bindtextdomain`, `textdomain` and plural Gettext calls. Two applications may
both use the `messages` domain with different catalogs. Locale state survives
shutdown callbacks and destructors, then is restored before the next request.

MO catalogs are immutable shared cache entries, accounted to MemCP's cache
budget and invalidated on a later request when file size or modification time
changes. Catalogs are limited to 16 MiB on disk and a conservative 64 MiB parsed
allocation estimate. The loader validates string ranges and expansion before
parsing. A request keeps its catalog references until cleanup. This shared Go
cache is separate from PHP's Zend allocation quota.

Native IMAP/c-client must **not** be loaded into the embedded ZTS runtime.
Configure `(settings "PHPIMAPBinary" "/usr/bin/php")`, or the corresponding
PHP dashboard field, with an NTS PHP CLI that has native IMAP installed. Restart
MemCP afterwards. An empty value disables the IMAP adapter. On Ubuntu, the
helper can use the distribution's `php-cli` and `php-imap` packages; its extension
ABI is independent of the embedded PHP build.

The adapter exposes the helper's IMAP functions and constants, and reports the
`imap` extension as loaded. It forwards positional/named arguments, binary
strings, arrays, objects and `imap_savebody` stream output. Function reflection
shows the adapter's variadic signature. Only IMAP calls run in a lazily started
helper process, one per request; PHP application code and PDO queries keep the
direct in-process MemCP bridge. Separate helpers isolate c-client global state
and contain a native IMAP crash. An individual call is bounded to 60 seconds,
and each protocol frame to 64 MiB. The helper inherits the configured PHP
request memory ceiling.

Explicit `imap_close()` and normal object destruction close handles. During
request shutdown, leftover connections are closed by the helper's CLI shutdown;
MemCP allows 200 ms for cleanup, then kills and reaps a stuck helper. The request
retains helper ownership through PHP's final object cleanup, including an early
`exit`, uncaught exception or Zend memory-limit failure. Other in-process native
extensions still require ZTS support: arbitrary native memory corruption cannot
be contained inside the same process.

## Application extension build

Use the same ZTS `php-config` for all extensions loaded into embedded PHP.
The PHP CI workflow builds and tests `gettext`, PDO/MySQL, XML/DOM, Imagick,
Intl, ZIP, JSON, Session, GMP, mbstring, cURL and OpenSSL, plus IMAP through
its separate NTS helper. It also checks `mb_regex_encoding`: building mbstring
with `--disable-mbregex` prevents mPDF from starting.

The relevant PHP configure options are:

```sh
--with-gettext --enable-mbstring --enable-intl --with-zip --with-gmp \
--with-curl --with-openssl --with-zlib --with-iconv \
--enable-dom --enable-xml --enable-xmlreader --enable-xmlwriter --enable-simplexml \
--enable-pdo --with-pdo-mysql=mysqlnd --enable-mysqlnd --enable-session
```

Install the development headers for Oniguruma, libzip, ICU, GMP, libxml2,
cURL, OpenSSL and ImageMagick. Build the released Imagick extension separately
using the ZTS installation's `phpize` and `--with-php-config=...`, then enable
`extension=imagick.so` in that installation's `php.ini`. The CI workflow pins
PHP 8.5.10 and Imagick 3.8.1 downloads by checksum and keeps all external sources
outside this repository.

## PHP quotas, concurrency and tuning

Configure PHP globally through `(settings)` or the **PHP** group in the dashboard.
Changes take effect after restarting MemCP. The existing settings mechanism saves
values to `data/settings.json` on orderly shutdown and loads them on startup.
PHP tuning CLI flags are no longer supported.

| Setting | Default | Meaning |
| --- | --- | --- |
| `PHPIMAPBinary` | empty | NTS PHP CLI executable with native IMAP; empty disables the adapter. |
| `PHPThreads` | `4` | Maximum simultaneous PHP executions; excess requests wait. |
| `PHPMemoryLimit` | `1073741824` (1 GiB) | Per-request allocation ceiling including retained PDO results; minimum 8 MiB. |
| `PHPMaxWaitMilliseconds` | `30000` | Queue timeout before HTTP 503; `0` waits indefinitely. |
| `PHPOutputBuffer` | `4096` | Output buffer bytes; `0` disables buffering. |
| `PHPOpcacheMemory` | `536870912` (512 MiB) | Shared opcode cache bytes; minimum 32 MiB, rounded up to whole MiB. |

For example: `(settings "PHPMemoryLimit" 1073741824)` sets the request quota
to 1 GiB. The dashboard accepts `1024MiB` or `1GiB` in this field and stores
numeric bytes. Existing memory budget fields also accept sizes such as `10MB`
(10,000,000 bytes) and `10MiB` (10,485,760 bytes). Save by leaving the field or
pressing Enter. Invalid sizes are rejected without changing the setting.

MemCP sets both `memory_limit` and PHP 8.5's startup-only
[`max_memory_limit`](https://www.php.net/manual/en/ini.core.php#ini.max-memory-limit)
to the selected quota. A script can lower its limit but cannot raise it above
the ceiling or disable it with `ini_set()`. These host settings override the
corresponding external `php.ini` values. Zend's allocator must remain enabled;
`USE_ZEND_ALLOC` must be unset or `1`; other values, including an empty value,
are rejected.

The quota includes unfetched PDO statement buffers, not just strings returned
by `fetch()`. Results transfer to the PHP request heap after the Go call returns;
an allocation failure frees the temporary bridge buffer before PHP aborts the
request. PDO teardown rolls back unfinished transactions. The authenticated
in-process SQL bridge remains in use; no MySQL socket round trip is introduced.

This is an allocation quota, **not a process-wide RAM limit or tenant sandbox**.
At four threads and 1 GiB, PHP request heaps may total about 4 GiB, plus the
shared opcode cache, interpreter overhead, transient bridge buffers, and MemCP
storage/query memory. The bridge retains its 64 MiB per-result bound during
transfer. Native extension allocations and child processes, such as an external
`php dbcheck.php`, are outside this quota. An OS memory limit on this shared
process also limits MemCP and can terminate the database process.

OPcache is enabled with room for 20,000 scripts and a 16 MiB interned-string
buffer. PHP JIT is disabled. Timestamp validation stays enabled on every request
(`opcache.revalidate_freq=0`) so redeployments do not require disabling cache
validation or restarting the database. Output buffering batches small writes;
streaming endpoints can finish their output buffers and call `flush()` as usual.
Other PHP settings and extension loading remain in the external `php.ini`.

## Per-directory routing and access rules

The PHP host reads `.htaccess` before serving either PHP or static files.
Rules are cached using the same directory watcher as Scheme's `(watch)`;
file changes, atomic replacements, removal and recreation invalidate the cache.
Parent-directory rules run before descendant rules.

Supported directives are `RewriteEngine`, `RewriteCond %{REQUEST_FILENAME}`
with `-f`, `!-f`, `-d`, or `!-d`, and `RewriteRule`. Supported flags are `NC`,
`L`, `END`, `QSA`, `B`, `UnsafeAllow3F`, `F`, and `G`. Internal rewrites preserve
PHP's original request URI. `B` escapes captured query characters; `QSA` retains
the incoming query. Invalid or unsupported directives and read errors return
HTTP 500, so a configuration error cannot silently disable access rules.
This is a rewrite subset, not an Apache configuration interpreter: directives
such as `Options`, `Require`, and `<Files>` must be expressed using the supported
rules before hosting that application.

## Foployment deployment

Use an external PHP ZTS installation with the application's extensions,
including `gettext`, `mbstring`, `intl`, `gd`, and `pdo_mysql`. Foployment also
executes `php dbcheck.php`, so its command-line PHP must have the required
extensions and be available on the server's `PATH`. Install `tar` and the archive
utilities required by the release format.

Place the unmodified `foployment/out` contents in `htdocs/foployment` and start:

```sh
./memcp-php --no-repl -data /srv/memcp-data --api-port=8080 \
  --mysql-port=3307 --serve /srv/htdocs lib/main.scm
```

In Foployment's `conf.json`, set `FOP.Database` to its dedicated MemCP database
and an administrative MemCP account. Set `FOP.Configuration.path` to `"../"`,
`baseurl` to `"http://localhost:8080"`, and `Baseurl` to
`"http://localhost:8080/foployment/"`. `FOP.Configuration.dbhost` becomes the
child application's database host; for a nonstandard local MySQL port, use
`"127.0.0.1;port=3307"`, since the generated child configuration does not copy
the parent's `Database.Port`. The child uses its own account and database,
created through `GRANT ALL ... IDENTIFIED BY`.

Configure the external embedded PHP's `php.ini` for the archive size, e.g.:

```ini
short_open_tag=Off
upload_max_filesize=256M
post_max_size=272M
max_execution_time=900
```

`short_open_tag=Off` also allows XML declarations in PHP templates. For a local
test deployment, set `FOP.Cron.cronmode` to `"none"` in both application configs
(use Foployment's child configuration editor before uploading), so the fixture
does not install scheduled jobs. Production cron configuration is an
application administration choice.

Open `/foployment/`, create its first user, create an instance with path `eur`,
and upload the original `release.tar.xz` through the instance form. The deployed
application is served at `/eur/`; each application's own `.htaccess` protects
its configuration and routes its virtual URLs. Enable MemCP `TracePrint` and
set `TracePrintMaxLength` to `0` when collecting an untruncated query log.

## PDO connections

The addon registers **only `memcp:`** and leaves other driver registrations
intact. It also routes a narrow set of local MySQL DSNs at base `PDO` connection
construction, before the original constructor selects a driver:

```php
$local = new PDO('memcp:dbname=shop', $user, $password);
$mysql = new PDO('mysql:host=db.example;dbname=shop', $user, $password);
$sqlite = new PDO('sqlite:/path/to/cache.sqlite');
```

A base `new PDO(...)` or `PDO::connect(...)` using
`mysql:host=127.0.0.1;port=3307;dbname=shop` automatically selects the RAM path
if 3307 is this process's successfully started, Scheme-registered MySQL port.
The exact hostname `localhost` also matches. Host, port and database must all
be explicit; optional `charset=utf8mb4` or `charset=utf8` is accepted.
The resulting PDO driver name is `memcp`, and the same username/password
authentication applies. The caller's DSN string is preserved.

Other hosts/ports, Unix sockets, omitted/ambiguous DSN fields, other charsets,
driver-specific options, timeouts, native-prepare requests and persistent connections
retain native PDO behavior. PDO subclasses (including `Pdo\Mysql`) retain
their original behavior too. The constructor dispatch is installed once during
PHP module initialization, before request threads start; no handler pointers
change while requests execute. No external source is patched.

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

Results are limited to 64 MiB and copied through a temporary C-owned buffer
into the PHP request heap, where they count against the memory quota. Existing
PDO statements retain their own data when another query reuses
the collector; fetching rows never calls back into Go. Strings preserve arbitrary
bytes. Use SQL pagination for larger results. The 30-second query timeout,
process-list tracking, cancellation and transaction handling remain active.
Closing a connection rolls back unfinished transactions, releases its locks and
unregisters its session, including during request teardown after PHP exceptions.

Applications or ORMs that recognize only specific PDO driver names may need a
MemCP/MySQL-dialect adapter. Applications using `mysqli`, including unmodified
WordPress, can instead use MemCP's existing MySQL TCP/Unix socket frontend.
That path uses the MySQL protocol even though PHP runs in the same process.
The normal MySQL host/port/username/password installer fields work with this
wire frontend; neither application needs a MemCP-specific DSN. PDO constructor
routing does **not** accelerate `mysqli`: that additionally requires integration
with mysqlnd.

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

For extension/IMAP integration coverage with a local build:

```sh
MEMCP_TEST_IMAP_BINARY=/usr/bin/php MEMCP_TEST_PHP_EXTENSIONS=1 \
  make test-php PHP_CONFIG=/path/to/php-zts/bin/php-config
```

The IMAP tests use a private local mock mailbox; no external mail account is required.
