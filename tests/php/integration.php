<?php
// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

function check($condition, $message) {
    if (!$condition) throw new RuntimeException($message);
}
function fails($callback, $state = null) {
    try { $callback(); } catch (PDOException $e) {
        if ($state !== null) check(($e->errorInfo[0] ?? $e->getCode()) === $state, 'SQLSTATE: ' . ($e->errorInfo[0] ?? $e->getCode()));
        return;
    }
    throw new RuntimeException('Expected PDOException at line ' . debug_backtrace()[0]['line']);
}
header('Content-Type: application/json');
try {
    $action = $_GET['action'] ?? 'probe';
    if ($action === 'routing') {
        echo json_encode(['script' => $_SERVER['SCRIPT_NAME'], 'path_info' => $_SERVER['PATH_INFO'] ?? '', 'uri' => $_SERVER['REQUEST_URI'], 'method' => $_SERVER['REQUEST_METHOD'], 'post' => $_POST['value'] ?? '', 'cookie' => $_COOKIE['probe'] ?? '', 'header' => $_SERVER['HTTP_X_PROBE'] ?? '']);
        return;
    }
    if ($action === 'probe') {
        echo json_encode(['zts' => PHP_ZTS, 'opcache' => opcache_get_status(false) !== false,
            'drivers' => PDO::getAvailableDrivers()]);
        return;
    }
    check(!isset($GLOBALS['previous_request']), 'Request globals leaked');
    $GLOBALS['previous_request'] = true;
    $db = new PDO('memcp:dbname=memcp-tests', 'root', 'admin', [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]);
    if ($action === 'setup') {
        $db->exec('CREATE TABLE php_test (id INT PRIMARY KEY AUTO_INCREMENT, value TEXT)');
        $db->exec("CREATE USER php_reader IDENTIFIED BY 'reader-password'");
     } elseif ($action === 'quota') {
        check((int)ini_get('max_memory_limit') === 32*1024*1024, 'Host memory ceiling');
        @ini_set('max_memory_limit', '-1');
        @ini_set('memory_limit', '-1');
        check((int)ini_get('memory_limit') === 32*1024*1024, 'Unlimited memory bypass');
        @ini_set('memory_limit', '64M');
        check((int)ini_get('memory_limit') === 32*1024*1024, 'Raised memory bypass');
        check((int)ini_get('output_buffering') === 4096, 'Output buffering preset');
        $before = memory_get_usage();
        $held = $db->query("SELECT REPEAT('q', 1048576) AS payload");
        check(memory_get_usage() - $before >= 1048576, 'Unfetched PDO result must count against quota');
        $held->closeCursor();
        check(memory_get_usage() - $before < 131072, 'Closed PDO cursor releases its quota');
    } elseif ($action === 'oom-php' || $action === 'oom-pdo') {
        $db->beginTransaction();
        $db->exec("INSERT INTO php_test(value) VALUES ('oom rollback')");
        if ($action === 'oom-php') { $huge = str_repeat('x', 64*1024*1024); }
        $held = [];
        for ($i = 0; $i < 64; $i++) { $held[] = $db->query("SELECT REPEAT('q', 1048576) AS payload"); }
        throw new RuntimeException('Memory ceiling was not enforced');
    } elseif ($action === 'pdo') {
        $reader = new PDO('memcp:dbname=memcp-tests', 'php_reader', 'reader-password');
        fails(fn() => $reader->query('SELECT COUNT(*) FROM php_test'));
        fails(fn() => $reader->exec("INSERT INTO php_test(value) VALUES ('forbidden')"));
        $db->exec('GRANT SELECT ON `memcp-tests`.* TO php_reader');
        check($reader->query('SELECT COUNT(*) FROM php_test')->fetchColumn() == 0, 'Read permission');
        $sqlite = new PDO('sqlite::memory:');
        check($sqlite->query('SELECT 42')->fetchColumn() == 42, 'SQLite alongside MemCP');
        check($db->getAttribute(PDO::ATTR_DRIVER_NAME) === 'memcp', 'PDO driver name');
        $stmt = $db->prepare('SELECT :s AS value, :n AS number, :b AS flag, :z AS nothing');
        $bytes = "hi'\\\0\xff";
        $stmt->bindValue(':s', $bytes, PDO::PARAM_STR);
        $stmt->bindValue(':n', 42, PDO::PARAM_INT);
        $stmt->bindValue(':b', true, PDO::PARAM_BOOL);
        $stmt->bindValue(':z', null, PDO::PARAM_NULL);
        $stmt->execute();
        $row = $stmt->fetch(PDO::FETCH_ASSOC);
        check($row['value'] === $bytes && is_numeric($row['number']) && $row['number'] == 42 && $row['nothing'] === null, 'Typed/binary parameters: ' . gettype($row['number']) . ':' . $row['number'] . ' hex=' . bin2hex($row['value']) . ' null=' . gettype($row['nothing']));
        check($stmt->fetch() === false, 'End of cursor');
        $stmt = $db->prepare('SELECT ? AS value');
        foreach (['', 'first', "x'); DROP TABLE php_test; --"] as $value) {
            $stmt->execute([$value]);
            check($stmt->fetchColumn() === $value, 'Repeated execute and injection protection');
            $stmt->closeCursor();
        }
        fails(fn() => $db->prepare('SELECT ?')->execute([]), 'HY093');
        fails(fn() => $db->prepare('SELECT :missing')->execute(), 'HY093');
        fails(fn() => $db->prepare('SELECT :mixed, ?'));
        fails(fn() => $db->prepare('SELECT 1', [PDO::ATTR_CURSOR => PDO::CURSOR_SCROLL]));
        fails(fn() => $db->setAttribute(PDO::ATTR_EMULATE_PREPARES, false));
        fails(fn() => $db->query('SELECT absent FROM php_missing_table'));
        fails(fn() => new PDO('memcp:dbname=memcp-tests', 'root', 'wrong'), '28000');
        fails(fn() => new PDO('memcp:dbname=memcp-tests', 'root', ''), '28000');
        fails(fn() => new PDO('memcp:dbname=memcp-tests'), '28000');
        fails(fn() => new PDO('memcp:dbname=memcp-tests', 'php_unknown', 'admin'), '28000');
        // An authenticated reader must not inherit root's transaction session.
        $db->beginTransaction();
        check($db->inTransaction() && !$reader->inTransaction(), 'PDO sessions are independent');
        $db->rollBack();
        fails(fn() => new PDO('memcp:dbname=does-not-exist', 'root', 'admin'), '3D000');
        fails(fn() => new PDO('memcp:host=elsewhere', 'root', 'admin'));
        fails(fn() => new PDO('memcp:dbname=memcp-tests', 'root', 'admin', [PDO::ATTR_PERSISTENT => true]));
        $stmt = $db->query('SELECT id, value FROM php_test WHERE id = -1');
        check($stmt->columnCount() === 2 && $stmt->fetch() === false, 'Empty-result metadata');
        $typed = $db->query("SELECT NULL AS n, CAST('42' AS SIGNED) AS i, CAST(12 AS CHAR CHARACTER SET utf8) AS s")->fetch(PDO::FETCH_ASSOC);
        check($typed === ['n' => null, 'i' => 42, 's' => '12'], 'Compiler types drive RAM packing');
        $typed = $db->query("SELECT 1 AS x, 'text' AS x")->fetch(PDO::FETCH_NUM);
        check($typed === [1, 'text'], 'Duplicate aliases retain positional types');
        $typed = $db->query("SELECT NULL AS x UNION ALL SELECT CAST('42' AS SIGNED)")->fetchAll(PDO::FETCH_NUM);
        check($typed === [[null], [42]], 'UNION types apply after initial NULL');
        $db->beginTransaction();
        check($db->inTransaction(), 'BEGIN');
        $db->exec("INSERT INTO php_test (value) VALUES ('rollback')");
        $db->rollBack();
        check(!$db->inTransaction(), 'ROLLBACK');
        check($db->query('SELECT COUNT(*) FROM php_test')->fetchColumn() == 0, 'Rollback undoes insert');
        $db->beginTransaction();
        check($db->exec("INSERT INTO php_test (value) VALUES ('committed')") === 1, 'Affected rows');
        check((int)$db->lastInsertId() > 0, 'Insert ID');
        $db->commit();
        check($db->query('SELECT COUNT(*) FROM php_test')->fetchColumn() == 1, 'Commit persists');
    } elseif ($action === 'wire') {
        $config = json_decode(file_get_contents(__DIR__ . '/wire.json'), true);
        $wire = new PDO('mysql:unix_socket=' . $config['socket'] . ';dbname=memcp-tests', 'root', 'admin', [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]);
        check($wire->getAttribute(PDO::ATTR_DRIVER_NAME) === 'mysql', 'PDO mysql dispatch');
        $typed = $wire->query("SELECT NULL AS n, CAST('42' AS SIGNED) AS i, CAST(12 AS CHAR CHARACTER SET utf8) AS s")->fetch(PDO::FETCH_ASSOC);
        check($typed === ['n' => null, 'i' => 42, 's' => '12'], 'Compiler types drive wire packing');
        $empty = $wire->query("SELECT CAST(NULL AS SIGNED) AS i FROM php_test WHERE id=-1");
        check($empty->getColumnMeta(0)['native_type'] === 'LONGLONG' && $empty->fetch() === false, 'Typed empty wire header');
        $wire->setAttribute(PDO::ATTR_EMULATE_PREPARES, false);
        $prepared = $wire->prepare('SELECT ? AS value');
        foreach ([[42, PDO::PARAM_INT], ['007', PDO::PARAM_STR], [null, PDO::PARAM_NULL], [17, PDO::PARAM_INT]] as [$value, $type]) {
            $prepared->bindValue(1, $value, $type);
            $prepared->execute();
            check($prepared->fetchColumn() === $value, 'Native parameter type changes invalidate compiler metadata');
            $prepared->closeCursor();
        }
        $wire->setAttribute(PDO::ATTR_EMULATE_PREPARES, true);
        for ($i = 0; $i < 30; $i++) {
            check($wire->query('SELECT 1')->fetchColumn() == 1, 'PDO wire query');
            check($db->query('SELECT 2')->fetchColumn() == 2, 'PDO direct query alongside wire');
        }
    } elseif ($action === 'route-dsn') {
        $config = json_decode(file_get_contents(__DIR__ . '/wire.json'), true);
        foreach (['localhost', '127.0.0.1'] as $host) {
            $dsn = 'mysql:host=' . $host . ';port=' . $config['port'] . ';dbname=memcp-tests;charset=utf8mb4';
            $before = $dsn;
            foreach ([
                fn() => new PDO($dsn, 'root', 'admin'),
                fn() => PDO::connect(dsn: $dsn, username: 'root', password: 'admin')
            ] as $open) {
                $local = $open();
                check($local->getAttribute(PDO::ATTR_DRIVER_NAME) === 'memcp', 'Local PDO DSN routes in-process');
                check($local->query('SELECT 1')->fetchColumn() == 1, 'Routed PDO result');
                check($dsn === $before, 'Caller DSN must remain unchanged');
            }
            fails(fn() => new PDO($dsn, 'root', 'wrong'), '28000');
            fails(fn() => PDO::connect($dsn), '28000');
            $reader = new PDO($dsn, 'php_reader', 'reader-password');
            check($reader->getAttribute(PDO::ATTR_DRIVER_NAME) === 'memcp', 'Non-root route');
        }
        $dsn = 'mysql:host=127.0.0.1;port=' . $config['other_port'] . ';dbname=memcp-tests';
        $wire = new PDO($dsn, 'root', 'admin');
        check($wire->getAttribute(PDO::ATTR_DRIVER_NAME) === 'mysql', 'Other TCP port remains wire');
        check($wire->query('SELECT 2')->fetchColumn() == 2, 'Other port wire query');
        $dsn = 'mysql:host=127.0.0.1;port=' . $config['port'] . ';dbname=memcp-tests';
        $wire = new PDO($dsn, 'root', 'admin', [PDO::ATTR_EMULATE_PREPARES => false]);
        check($wire->getAttribute(PDO::ATTR_DRIVER_NAME) === 'mysql', 'Native prepare option keeps wire');
        check(!$wire->getAttribute(PDO::ATTR_EMULATE_PREPARES), 'Native prepare option preserved');
        $wire->setAttribute(PDO::ATTR_EMULATE_PREPARES, true);
        check($wire->query('SELECT 3')->fetchColumn() == 3, 'Wire query after enabling emulation');
        $wire = Pdo\Mysql::connect($dsn, 'root', 'admin');
        check($wire->getAttribute(PDO::ATTR_DRIVER_NAME) === 'mysql', 'Driver-specific PDO class keeps wire');
    } elseif ($action === 'buffers') {
        $first = $db->query("SELECT 'first' AS value, NULL AS optional UNION ALL SELECT 'second', 'present'");
        $second = $db->query("SELECT 'different' AS other");
        check($first->fetch(PDO::FETCH_ASSOC) === ['value'=>'first', 'optional'=>null], 'Buffered statement survives another query');
        check($second->fetch(PDO::FETCH_ASSOC) === ['other'=>'different'], 'Different metadata');
        check($first->fetch(PDO::FETCH_ASSOC) === ['value'=>'second', 'optional'=>'present'], 'Buffered next row');
        $large = $db->query("SELECT REPEAT('x', 131072) AS payload");
        fails(fn() => $db->query('SELECT missing FROM php_missing_table'));
        check($db->query('SELECT 1')->fetchColumn() == 1, 'Reuse after query error');
        check(strlen($large->fetchColumn()) === 131072, 'Large result survives reuse and error');
        check($db->query('SELECT value FROM php_test WHERE id = -1')->fetchAll() === [], 'Empty result after large result');
        $stmt = $db->prepare('SELECT ? AS bytes');
        foreach (["a\0b", '', 'last'] as $value) {
            $stmt->execute([$value]); check($stmt->fetchColumn() === $value, 'Reused binary result'); $stmt->closeCursor();
        }
    } elseif ($action === 'latency') {
        // Perf-capable fixture: fixed query, warm connection, timer inside PHP.
        $warmup = min(1000, max(1, (int)($_GET['warmup'] ?? 100)));
        $samples = min(100000, max(1, (int)($_GET['samples'] ?? 200)));
        $times = [];
        for ($i = 0; $i < $warmup + $samples; $i++) {
            $start = hrtime(true); $stmt = $db->query('SELECT 1'); $value = $stmt->fetchColumn(); $stmt->closeCursor();
            $elapsed = hrtime(true) - $start; check($value == 1, 'Latency result');
            if ($i >= $warmup) $times[] = $elapsed;
        }
        echo json_encode(['ok'=>true, 'ns'=>$times]); return;
    } elseif ($action === 'abandon') {
        $db->beginTransaction();
        $db->exec("INSERT INTO php_test (value) VALUES ('abandoned')");
        // PDO teardown must roll back, including on a PHP exception.
        throw new RuntimeException('intentional request error');
    } elseif ($action === 'verify') {
        check($db->query("SELECT COUNT(*) FROM php_test WHERE value = 'abandoned'")->fetchColumn() == 0, 'Request teardown rollback');
    } elseif ($action === 'parallel') {
        $config = json_decode(file_get_contents(__DIR__ . '/wire.json'), true);
        $db = new PDO('mysql:host=127.0.0.1;port=' . $config['port'] . ';dbname=memcp-tests', 'root', 'admin');
        check($db->getAttribute(PDO::ATTR_DRIVER_NAME) === 'memcp', 'Parallel routed PDO');
        $stmt = $db->prepare('SELECT ? AS value');
        $value = $_GET['value'];
        $stmt->execute([$value]);
        $start = microtime(true);
        usleep(250000);
        $end = microtime(true);
        check($stmt->fetchColumn() === $value, 'Connections leaked state across threads');
        echo json_encode(['ok' => true, 'value' => $value, 'start' => $start, 'end' => $end]);
        return;
    }
    echo json_encode(['ok' => true]);
} catch (Throwable $e) {
    http_response_code(500);
    echo json_encode(['error' => $e->getMessage(), 'line' => $e->getLine()]);
}
