<?php
// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later
namespace IMAP {
    final class Connection {
        private int $handle;
        private function __construct(int $handle) { $this->handle = $handle; }
        private function __clone() {}
        public function __serialize(): array { throw new \Exception('Serialization of IMAP\\Connection is not allowed'); }
        public function __unserialize(array $data): void { throw new \Exception('Unserialization of IMAP\\Connection is not allowed'); }
        public static function fromHandle(int $handle): self { return new self($handle); }
        public function isOpen(): bool { return $this->handle !== 0; }
        public function handle(): int { if (!$this->handle) throw new \ValueError('IMAP connection is already closed'); return $this->handle; }
        public function closed(): void { $this->handle = 0; }
        public function __destruct() {
            if ($this->handle && !\memcp_imap_shutting_down()) {
                try { \MemCP\IsolatedIMAP\dispatch('imap_close', [$this]); } catch (\Throwable $ignored) {}
                $this->handle = 0;
            }
        }
    }
}
namespace MemCP\IsolatedIMAP {
    function packValue($value, array &$streams, int $depth = 0): array {
        if ($depth > 128) throw new \ValueError("IMAP argument nesting exceeds 128 levels");
        if ($value instanceof \IMAP\Connection) return ['h', $value->handle()];
        if (is_resource($value)) {
            if (get_resource_type($value) !== 'stream') throw new \TypeError('Unsupported IMAP resource');
            $id = count($streams); $streams[$id] = $value; return ['stream', $id];
        }
        if (is_string($value)) return ['s', base64_encode($value)];
        if (is_array($value)) {
            $out = [];
            foreach ($value as $key => $item) $out[] = [packValue($key, $streams), packValue($item, $streams, $depth + 1)];
            return ['a', $out];
        }
        if (is_object($value)) throw new \TypeError('Unsupported IMAP object');
        return ['v', $value];
    }
    function unpackValue(array $node) {
        switch ($node[0]) {
            case 's': return base64_decode($node[1], true);
            case 'v': return $node[1];
            case 'h': return \IMAP\Connection::fromHandle($node[1]);
            case 'o':
            case 'a':
                $out = [];
                foreach ($node[1] as $entry) $out[unpackValue($entry[0])] = unpackValue($entry[1]);
                return $node[0] === 'o' ? (object)$out : $out;
        }
        throw new \RuntimeException('Invalid IMAP helper response');
    }
    function dispatch(string $function, array $args) {
        if ($function === 'imap_is_open' && count($args) === 1 && ($args[0] ?? $args['imap']) instanceof \IMAP\Connection && !($args[0] ?? $args['imap'])->isOpen()) return false;
        $streams = []; $packed = [];
        foreach ($args as $key => $value) $packed[$key] = packValue($value, $streams);
        $reply = json_decode(\memcp_imap_rpc(json_encode(['function'=>$function, 'args'=>$packed, 'cwd'=>getcwd()], JSON_THROW_ON_ERROR)), true, 512, JSON_THROW_ON_ERROR);
        foreach ($reply['warnings'] ?? [] as [$level, $message]) trigger_error($message, E_USER_WARNING);
        if (isset($reply['error'])) {
            $class = in_array($reply['error'], [\ValueError::class, \TypeError::class, \ArgumentCountError::class], true) ? $reply['error'] : \RuntimeException::class;
            throw new $class($reply['message']);
        }
        foreach ($reply['writes'] ?? [] as $id => $data) {
            $bytes = base64_decode($data, true);
            while ($bytes !== '') {
                $count = fwrite($streams[$id], $bytes);
                if ($count === false || $count === 0) throw new \RuntimeException('Unable to write IMAP body to stream');
                $bytes = substr($bytes, $count);
            }
        }
        $result = unpackValue($reply['result']);
        if ($function === 'imap_close' && $result === true) ($args[0] ?? $args['imap'])->closed();
        return $result;
    }
}
