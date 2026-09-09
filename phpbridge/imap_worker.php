<?php
// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
// Private length-framed protocol. Only IMAP calls are accepted; no application
// PHP code or SQL runs in this helper. Process lifetime is one host request.
$handles = [];
$nextHandle = 1;
function readBytes($length) {
    $data = '';
    while (strlen($data) < $length) {
        $part = fread(STDIN, $length - strlen($data));
        if ($part === false || $part === '') throw new RuntimeException('IMAP protocol EOF');
        $data .= $part;
    }
    return $data;
}
function unpackValue($node, &$streams) {
    global $handles;
    switch ($node[0]) {
        case 's': return base64_decode($node[1], true);
        case 'v': return $node[1];
        case 'h':
            if (!isset($handles[$node[1]])) throw new ValueError('IMAP connection is already closed');
            return $handles[$node[1]];
        case 'a':
            $out = [];
            foreach ($node[1] as $entry) $out[unpackValue($entry[0], $streams)] = unpackValue($entry[1], $streams);
            return $out;
        case 'stream':
            $stream = fopen('php://temp', 'w+b');
            $streams[$node[1]] = $stream;
            return $stream;
    }
    throw new ValueError('Unsupported IMAP protocol argument');
}
function packValue($value) {
    global $handles, $nextHandle;
    if ($value instanceof \IMAP\Connection || is_resource($value)) {
        $id = $nextHandle++; $handles[$id] = $value;
        return ['h', $id];
    }
    if (is_string($value)) return ['s', base64_encode($value)];
    if (is_array($value) || is_object($value)) {
        $out = [];
        foreach ($value as $key => $item) $out[] = [packValue($key), packValue($item)];
        return [is_object($value) ? 'o' : 'a', $out];
    }
    return ['v', $value];
}
while (!feof(STDIN)) {
    $prefix = fread(STDIN, 4);
    if ($prefix === '' || $prefix === false) break;
    if (strlen($prefix) < 4) $prefix .= readBytes(4-strlen($prefix));
    $length = unpack('N', $prefix)[1];
    if ($length > 67108864) throw new RuntimeException('IMAP request exceeds protocol limit');
    $request = json_decode(readBytes($length), true, 512, JSON_THROW_ON_ERROR);
    $warnings = []; $streams = [];
    set_error_handler(function($severity, $message) use (&$warnings) { $warnings[] = [$severity, $message]; return true; });
    try {
        $function = $request['function'];
        if (!chdir($request['cwd'])) throw new RuntimeException('Unable to select IMAP working directory');
        if (strncmp($function, 'imap_', 5) !== 0 || !function_exists($function)) throw new ValueError('Unknown IMAP function');
        $args = [];
        foreach ($request['args'] as $key => $node) $args[$key] = unpackValue($node, $streams);
        $result = $function(...$args);
        if ($function === 'imap_close' && $result === true) unset($handles[($request['args'][0] ?? $request['args']['imap'])[1]]);
        $writes = [];
        foreach ($streams as $id => $stream) { rewind($stream); $writes[$id] = base64_encode(stream_get_contents($stream)); }
        $reply = ['result' => packValue($result), 'writes' => $writes, 'warnings' => $warnings];
    } catch (Throwable $error) {
        $reply = ['error' => get_class($error), 'message' => $error->getMessage(), 'warnings' => $warnings];
    } finally {
        foreach ($streams as $stream) fclose($stream);
        restore_error_handler();
    }
    $data = json_encode($reply, JSON_THROW_ON_ERROR | JSON_INVALID_UTF8_SUBSTITUTE);
    if (strlen($data) > 67108864) throw new RuntimeException('IMAP response exceeds protocol limit');
    $frame = pack('N', strlen($data)) . $data;
    while ($frame !== '') { $written = fwrite(STDOUT, $frame); if (!$written) exit(1); $frame = substr($frame, $written); }
    fflush(STDOUT);
    unset($args, $request, $result, $reply, $data, $frame, $streams, $writes, $warnings, $error);
}
// The normal CLI shutdown closes retained IMAP objects. The host also bounds
// shutdown time and reaps this process, including when cleanup itself hangs.
