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
        echo json_encode(['script' => $_SERVER['SCRIPT_NAME'], 'path_info' => $_SERVER['PATH_INFO'] ?? '', 'uri' => $_SERVER['REQUEST_URI']]);
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
