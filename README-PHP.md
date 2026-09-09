<!-- Copyright (C) 2026 Carl-Philip Hänsch -->

# Embedded PHP

MemCP can optionally host PHP applications in its own process using the
versioned FrankenPHP Go library. The host is application-independent: choose
an application's document root, listen address, thread count, and optional
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

## Serve an application

```sh
./memcp-php --no-repl -data /path/to/persistent/memcp-data \
  --php-root=/path/to/application/public \
  --php-listen=127.0.0.1:8080 --php-threads=4 \
  --php-front-controller=index.php lib/main.scm
```

Use the application's actual public directory. Without `--php-front-controller`,
missing files return 404. Existing directories use `index.php` or `index.html`; PHP `PATH_INFO` is supported. Static files
are served from the document root. Dotfiles, PHP source backups, and symlinks
outside the root are rejected. The PHP listener is separate from the SQL API;
it binds to loopback by default. Configure an HTTPS reverse proxy when exposing
it remotely. SIGTERM/SIGINT drains PHP before shutting down storage.

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

Results are buffered with a 64 MiB limit per result and copied into C-owned
memory in one batch. Strings retain arbitrary bytes. Use SQL pagination for
larger results. SQL execution has a 30-second timeout. Connections have separate
sessions, appear in the process list, and roll back unfinished transactions on
close. PDO teardown also runs after PHP exceptions. This is an initial driver,
not a claim of compatibility with every framework-specific PDO attribute.

Applications or ORMs that recognize only specific PDO driver names may need a
MemCP/MySQL-dialect adapter. Applications using `mysqli`, including unmodified
WordPress, can instead use MemCP's existing MySQL TCP/Unix socket frontend.
That path uses the MySQL protocol even though PHP runs in the same process.
A native PDO driver alone cannot transparently replace `mysqli`.

## Verification

Install `pdo_sqlite` alongside PDO in the ZTS build, then run:

```sh
make test-php PHP_CONFIG=/path/to/php-zts/bin/php-config
make test
```

`test-php` creates a disposable server and database. It checks ZTS/OPcache,
coexistence with SQLite, authentication failures, binary parameters, request
teardown rollback, independent parallel requests, and HTTP file boundaries.
It contains only our own PHP test fixtures and does not download applications.

For a WordPress demonstration, download WordPress and WP-CLI into a directory
outside the checkout. Configure a dedicated persistent MemCP database and
Unix socket in `wp-config.php`, set the PHP document root to the WordPress
directory, and run the normal WordPress installer. Keep credentials, uploaded
files, and the MemCP data directory outside the source tree.
