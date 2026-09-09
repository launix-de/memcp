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
        for ($i = 0; $i < 30; $i++) {
            check($wire->query('SELECT 1')->fetchColumn() == 1, 'PDO wire query');
            check($db->query('SELECT 2')->fetchColumn() == 2, 'PDO direct query alongside wire');
        }
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
    echo json_encode(['error' => $e->getMessage()]);
}
