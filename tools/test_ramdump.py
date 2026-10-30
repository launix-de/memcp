#!/usr/bin/env python3
# Copyright (C) 2026 Carl-Philip Hänsch
# SPDX-License-Identifier: GPL-3.0-or-later
"""Focused stdlib-only checks: python3 -m unittest discover -s tools -p test_ramdump.py."""
import copy
import importlib.util
import io
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import unittest
from contextlib import nullcontext
from unittest.mock import patch

SCRIPT = Path(__file__).with_name('ramdump.py')
SPEC = importlib.util.spec_from_file_location('ramdump', SCRIPT)
dump = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(dump)


@unittest.skipUnless(sys.platform == 'linux', 'Linux /proc memory dumper')
class RamdumpTests(unittest.TestCase):
    start = 0x10000000
    identity = {'boot_id': 'fixture', 'start_ticks': 42, 'started_unix': 42,
                'executable_sha256': 'fixture'}

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='ramdump-test-')
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)

    def events(self, data, resume=None, unreadable=None):
        maps = '{:x}-{:x} rw-p 00000000 00:00 0 [heap]\n'.format(
            self.start, self.start + len(data))
        maps += 'ffffffffff600000-ffffffffff601000 r-xp 00000000 00:00 0 [vsyscall]\n'

        class Memory:
            def __enter__(self):
                return self

            def __exit__(self, *args):
                pass

            def fileno(self):
                return 123

        def proc_open(path, *args, **kwargs):
            if path.endswith('/mem'):
                return Memory()
            return io.BytesIO(maps.encode() if path.endswith('/maps') else b'')

        def pread(fd, size, address):
            if unreadable is not None and address <= unreadable < address + size:
                raise OSError(dump.errno.EIO, 'unreadable fixture page')
            return data[address - self.start:address - self.start + size]

        with patch('builtins.open', proc_open), \
                patch.object(dump, 'stopped_process', return_value=nullcontext()), \
                patch.object(dump, 'process_identity', return_value=self.identity), \
                patch.object(dump.os, 'pread', pread), \
                patch.object(dump.os, 'readlink', return_value='/fixture'), \
                patch.object(dump, 'capture_artifacts', return_value=iter(())):
            return list(dump.capture_events(42, resume=resume))

    def capture(self, data, path=None, continuing=False, refresh=False):
        path = path or self.root / 'dump'
        writer = dump.DumpWriter(path, None, continuing=continuing, refresh=refresh)
        try:
            if refresh:
                writer.prepare_refresh()
            events = self.events(data, writer.resume_request)
            for kind, payload in events:
                writer.accept(kind, payload)
            self.assertTrue(writer.finished)
            writer.manifest['complete'] = True
        finally:
            writer.close()
        return path, json.loads((path / 'manifest.json').read_text()), events

    def test_sparse_offsets_and_unreadable_pages(self):
        data = b'A' * 4096 + bytes(8192) + b'D' * 4096
        path = self.root / 'dump'
        writer = dump.DumpWriter(path, None)
        try:
            for kind, payload in self.events(data, unreadable=self.start + 4096):
                writer.accept(kind, payload)
        finally:
            writer.close()
        manifest = json.loads((path / 'manifest.json').read_text())
        region = manifest['mappings'][0]
        file = path / region['file']
        self.assertEqual(file.read_bytes(), data)
        self.assertLess(file.stat().st_blocks * 512, len(data))
        self.assertEqual(region['unreadable_ranges'], [
            {'start': hex(self.start + 4096), 'end': hex(self.start + 8192), 'errno': dump.errno.EIO}])
        self.assertEqual(manifest['bytes_zero'], 4096)
        self.assertEqual(manifest['mappings'][1]['skip_reason'], 'kernel_address')

    def test_compressed_protocol_elides_zeros_and_preserves_errors(self):
        events = list(dump.memory_events(self.start, b'A' * 4096 + bytes(409600), 4096))
        events.append((b'Z', b'{}'))
        wire = io.BytesIO()
        with patch.object(dump.sys, 'stdout', type('Output', (), {'buffer': wire})()):
            dump.send_stream(iter(events))
        self.assertLess(len(wire.getvalue()), 500)
        self.assertEqual(list(dump.receive_stream(io.BytesIO(wire.getvalue()))), events)
        with self.assertRaises(RuntimeError):
            list(dump.receive_stream(io.BytesIO(wire.getvalue()[:-1])))

        def failed():
            raise RuntimeError('original remote failure')
            yield

        wire = io.BytesIO()
        with patch.object(dump.sys, 'stdout', type('Output', (), {'buffer': wire})()):
            dump.send_stream(failed())
        with self.assertRaisesRegex(RuntimeError, 'original remote failure'):
            list(dump.receive_stream(io.BytesIO(wire.getvalue())))

    def test_resume_checkpoint_and_identity(self):
        data = b'A' * dump.CHUNK + b'B' * dump.CHUNK
        path = self.root / 'dump'
        writer = dump.DumpWriter(path, None)
        try:
            for kind, payload in self.events(data):
                writer.accept(kind, payload)
                if writer.done >= dump.CHUNK:
                    break
        finally:
            writer.close()
        initial = json.loads((path / 'manifest.json').read_text())
        self.assertEqual(initial['mappings'][0]['captured_until'], dump.CHUNK)
        writer = dump.DumpWriter(path, None, continuing=True)
        try:
            with self.assertRaisesRegex(RuntimeError, 'anderer Dump-Prozess'):
                dump.DumpWriter(path, None, continuing=True)
            bad = copy.deepcopy(writer.resume_request)
            bad['process_identity']['start_ticks'] += 1
            with self.assertRaisesRegex(RuntimeError, 'neu gestartet'):
                self.events(data, bad)
        finally:
            writer.close()
        path, manifest, events = self.capture(data, path, continuing=True)
        self.assertEqual((path / manifest['mappings'][0]['file']).read_bytes(), data)
        self.assertEqual(sum(len(p) - 8 for k, p in events if k == b'D'), dump.CHUNK)
        self.assertTrue(manifest['mixed_capture_times'])

    def test_refresh_implies_continue_and_repairs_same_size_corruption(self):
        data = b'A' * dump.CHUNK + b'B' * dump.CHUNK + b'C' * dump.CHUNK
        path, manifest, _ = self.capture(data)
        file = path / manifest['mappings'][0]['file']
        with file.open('r+b') as stream:
            stream.seek(32)
            stream.write(b'corrupt')
        current = b'A' * dump.CHUNK + bytes(dump.CHUNK) + b'C' * dump.CHUNK
        path, manifest, events = self.capture(current, path, refresh=True)
        self.assertEqual(file.read_bytes(), current)
        self.assertEqual(manifest['bytes_reused'], dump.CHUNK)
        self.assertEqual(manifest['bytes_refreshed'], 2 * dump.CHUNK)
        self.assertTrue(any(k == b'K' for k, _ in events))
        self.assertLess(file.stat().st_blocks * 512, file.stat().st_size)

    def test_changed_mapping_keeps_old_file(self):
        path, previous, _ = self.capture(b'A' * 8192)
        previous['complete'] = False
        (path / 'manifest.json').write_text(json.dumps(previous))
        changed = copy.deepcopy(previous)
        changed['started_at'] = '2026-10-30T12:00:00+00:00'
        region = changed['mappings'][0]
        region.update(permissions='r--p', status='pending', captured_until=0,
                      bytes_read=0, bytes_zero=0, unreadable_ranges=[])
        dump.resume_regions(changed, previous)
        self.assertEqual(changed['resume_changed_mappings'], 1)
        self.assertNotEqual(changed['mappings'][0]['file'], previous['mappings'][0]['file'])
        self.assertEqual((path / previous['mappings'][0]['file']).read_bytes(), b'A' * 8192)

    def test_real_capture_stop_and_ssh_reconstruction(self):
        code = '''import ctypes, time
libc = ctypes.CDLL(None)
if libc.prctl(0x59616d61, ctypes.c_ulong(-1), 0, 0, 0) != 0:
    raise SystemExit(77)
buffer = ctypes.create_string_buffer(b"ramdump-fixture")
print(ctypes.addressof(buffer), flush=True)
time.sleep(60)
'''
        target = subprocess.Popen([sys.executable, '-c', code], stdout=subprocess.PIPE, text=True)
        try:
            line = target.stdout.readline().strip()
            if not line:
                self.skipTest('test process cannot allow ptrace access')
            address = int(line)
            events = dump.capture_events(target.pid, stop=True)
            try:
                next(events)
                self.assertIn(dump.process_state(target.pid), ('T', 't'))
            finally:
                events.close()
            self.assertNotIn(dump.process_state(target.pid), ('T', 't'))
            bindir = self.root / 'bin'
            bindir.mkdir()
            ssh = bindir / 'ssh'
            ssh.write_text('#!/usr/bin/env python3\nimport os,sys\nos.execl("/bin/sh","sh","-c",sys.argv[3])\n')
            ssh.chmod(0o700)
            env = dict(os.environ, PATH=str(bindir) + ':' + os.environ['PATH'])
            path = self.root / 'remote'
            result = subprocess.run([sys.executable, str(SCRIPT), str(target.pid), '--ssh', 'fixture',
                                     '--stop', '-o', str(path), '--no-progress'],
                                    env=env, capture_output=True, text=True, timeout=30)
            self.assertEqual(result.returncode, 0, result.stderr)
            manifest = json.loads((path / 'manifest.json').read_text())
            self.assertTrue(manifest['complete'])
            self.assertTrue(manifest['executable_saved'])
            region = next(r for r in manifest['mappings'] if int(r['start'], 16) <= address < int(r['end'], 16))
            with (path / region['file']).open('rb') as stream:
                stream.seek(address - int(region['start'], 16))
                self.assertEqual(stream.read(16), b'ramdump-fixture\0')
        finally:
            # A failed pause assertion must not leave a stopped test process.
            if target.poll() is None:
                os.kill(target.pid, signal.SIGCONT)
                target.terminate()
            target.wait(timeout=10)
            target.stdout.close()


if __name__ == '__main__':
    unittest.main()
