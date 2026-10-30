#!/usr/bin/env python3
# Copyright (C) 2026 Carl-Philip Hänsch
# SPDX-License-Identifier: GPL-3.0-or-later
"""Dump Linux process memory into a local directory, optionally over SSH.

  ./tools/ramdump.py PID --ssh user@host [-o directory] [--stop]

Default directory: dump-PID/. One sparse file per readable mapping, named
START-END.bin in regions/. File offset = virtual address - START.
manifest.json records mappings, unreadable ranges and completion status.
The executable, mapped ELF libraries, installed separate debug files,
maps, smaps, status, auxv and cmdline are also saved. No symbols are downloaded.
--debug-dir adds a remote/local symbol search directory to /usr/lib/debug.
Remote mode needs Python 3 but creates no files on the remote host.

By default the target keeps running (a non-atomic live dump). --stop suspends
it with SIGSTOP during capture and resumes it afterwards unless already stopped.
This freezes threads but does not put the Go runtime at a GC safepoint.
Existing output directories require -c. Partial dumps are retained.
Use -c/--continue to resume an interrupted dump of the same process. Finished
regions are reused; partial regions resume at their last durable checkpoint.
Older dumps restart partial regions. Resumed dumps combine capture times.
Zero pages are omitted from transmission; SSH records use zlib compression.
--refresh implies -c and compares local memory blocks with the current process
using SHA-256, updating changed blocks. It also works on completed dumps.
"""
import argparse
import base64
import ctypes
import datetime
import errno
import fcntl
import hashlib
import json
import os
import platform
import shlex
import shutil
import signal
import stat
import struct
import subprocess
import sys
import threading
import time
import zlib
from contextlib import contextmanager
from pathlib import Path

MAGIC = b'RAMREGIONS4\n'
HEADER = struct.Struct('!cI')
ADDRESS = struct.Struct('!Q')
RANGE = struct.Struct('!QQ')
CHUNK = 1024 * 1024
READ_ERRORS = {errno.EIO, errno.EFAULT, errno.EPERM, errno.EACCES, errno.ENXIO}
PUNCH_HOLE = getattr(ctypes.CDLL(None, use_errno=True), 'fallocate', None)
if PUNCH_HOLE is not None:
    PUNCH_HOLE.argtypes = (ctypes.c_int, ctypes.c_int, ctypes.c_longlong, ctypes.c_longlong)
    PUNCH_HOLE.restype = ctypes.c_int


def now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def json_bytes(value):
    return json.dumps(value, ensure_ascii=True).encode('utf-8')


def sha256_file(source):
    source.seek(0)
    digest = hashlib.sha256()
    while True:
        data = source.read(CHUNK)
        if not data:
            return digest.hexdigest()
        digest.update(data)


def process_identity(pid):
    with open('/proc/{}/stat'.format(pid)) as source:
        ticks = int(source.read().rsplit(')', 1)[1].split()[19])
    with open('/proc/stat') as source:
        boot_time = next(int(line.split()[1]) for line in source if line.startswith('btime '))
    with open('/proc/sys/kernel/random/boot_id') as source:
        boot_id = source.read().strip()
    with open('/proc/{}/exe'.format(pid), 'rb') as source:
        executable_hash = sha256_file(source)
    return {'boot_id': boot_id, 'start_ticks': ticks,
            'started_unix': boot_time + ticks / os.sysconf('SC_CLK_TCK'),
            'executable_sha256': executable_hash}


def region_key(region):
    return tuple(region[key] for key in ('start', 'end', 'permissions', 'file_offset', 'device', 'inode', 'pathname'))


def region_file(region):
    """Validate address-based names, including reread generations on resume."""
    base = 'regions/{:016x}-{:016x}'.format(int(region['start'], 16), int(region['end'], 16))
    name = region['file']
    if name == base + '.bin':
        return name
    if name.startswith(base + '.') and name.endswith('.bin'):
        generation = name[len(base) + 1:-4]
        if len(generation) == 16 and all(c in '0123456789abcdef' for c in generation):
            return name
    raise RuntimeError('Ungültiger Mapping-Dateiname im Manifest')


def resume_regions(metadata, previous):
    """Reuse matching mappings; reread changed ranges of the same process."""
    if previous['pid'] != metadata['pid'] or previous['hostname'] != metadata['hostname']:
        raise RuntimeError('Fortsetzen verweigert: anderer Prozess oder Rechner')
    identity = metadata['process_identity']
    if previous.get('process_identity'):
        if previous['process_identity'] != identity:
            raise RuntimeError('Fortsetzen verweigert: Prozess wurde neu gestartet oder Binary geändert')
    else:
        captured = datetime.datetime.fromisoformat(previous['started_at']).timestamp()
        if identity['started_unix'] > captured or previous.get('legacy_executable_sha256') != identity['executable_sha256']:
            raise RuntimeError('Fortsetzen verweigert: alter Dump passt nicht zum laufenden Prozess')
    if previous.get('anonymous_only', False) != metadata['anonymous_only']:
        raise RuntimeError('Fortsetzen benötigt dieselbe --anonymous-Einstellung')
    current = {region_key(r): r for r in metadata['mappings'] if r['file']}
    superseded = list(previous.get('superseded_mappings', []))
    reused = set()
    changed = 0
    for saved in previous['mappings']:
        if not saved.get('file') or saved.get('captured_until', 0) == 0:
            continue
        key = region_key(saved)
        if key not in current:
            superseded.append(dict(saved, superseded_at=metadata['started_at'], reason='changed_or_unmapped'))
            changed += 1
            continue
        reused.add(key)
        for field in ('file', 'status', 'bytes_read', 'bytes_zero', 'unreadable_ranges', 'captured_until'):
            current[key][field] = saved[field]
    occupied = {r['file'] for r in superseded}
    generation = hashlib.sha256(metadata['started_at'].encode()).hexdigest()[:16]
    for key, region in current.items():
        if key not in reused and region['file'] in occupied:
            # Keep the old bytes intact if a new mapping occupies the same range.
            region['file'] = region['file'][:-4] + '.' + generation + '.bin'
    metadata['superseded_mappings'] = superseded
    metadata['resume_changed_mappings'] = changed
    if changed:
        metadata['warnings'].append('{} gespeicherte Mappings verändert/verschwunden; aktuelle Bereiche werden neu gelesen, alte Dateien bleiben erhalten.'.format(changed))
    metadata['capture_sessions'] = list(previous.get('capture_sessions', [previous['started_at']])) + [metadata['started_at']]
    metadata['started_at'] = previous['started_at']
    metadata['mixed_capture_times'] = True
    metadata['warnings'].append('Fortgesetzter Dump: wiederverwendete und neue Daten stammen aus unterschiedlichen Aufnahmezeiten.')
    metadata['bytes_read'] = sum(r['bytes_read'] for r in metadata['mappings'])
    metadata['bytes_zero'] = sum(r['bytes_zero'] for r in metadata['mappings'])
    metadata['bytes_unreadable'] = sum(int(h['end'], 16) - int(h['start'], 16)
                                       for r in metadata['mappings'] for h in r['unreadable_ranges'])
    if previous.get('refresh'):
        for region in metadata['mappings']:
            if region['file']:
                region.update(status='pending', captured_until=0, bytes_read=0, bytes_zero=0, unreadable_ranges=[])
        metadata.update(refresh=True, bytes_read=0, bytes_zero=0, bytes_unreadable=0,
                        bytes_reused=0, bytes_refreshed=0)


def memory_events(address, data, page):
    """Coalesce runs so zero pages cost one address/length record, not bytes."""
    zero = bytes(page)
    beginning = 0
    is_zero = data[:page] == zero[:min(page, len(data))]
    for offset in range(page, len(data), page):
        block = data[offset:offset + page]
        next_zero = block == zero[:len(block)]
        if next_zero != is_zero:
            yield (b'N', RANGE.pack(address + beginning, offset - beginning)) if is_zero else (
                b'D', ADDRESS.pack(address + beginning) + data[beginning:offset])
            beginning, is_zero = offset, next_zero
    yield (b'N', RANGE.pack(address + beginning, len(data) - beginning)) if is_zero else (
        b'D', ADDRESS.pack(address + beginning) + data[beginning:])


def process_state(pid):
    with open('/proc/{}/status'.format(pid)) as source:
        for line in source:
            if line.startswith('State:'):
                return line.split()[1]
    raise RuntimeError('Prozessstatus nicht verfügbar')


@contextmanager
def stopped_process(pid, stop):
    resume = False
    # An open proc stat FD does not retarget a reused PID.
    with open('/proc/{}/stat'.format(pid)) as identity:
        original = identity.read().rsplit(')', 1)[1].split()[19]
        try:
            if stop and process_state(pid) not in ('T', 't'):
                if pid == os.getpid():
                    raise RuntimeError('--stop kann das Dump-Skript selbst nicht anhalten')
                os.kill(pid, signal.SIGSTOP)
                resume = True
                deadline = time.monotonic() + 10
                while True:
                    tasks = os.listdir('/proc/{}/task'.format(pid))
                    if all(process_state('{}/task/{}'.format(pid, tid)) in ('T', 't') for tid in tasks):
                        break
                    if time.monotonic() >= deadline:
                        raise RuntimeError('Nicht alle Threads konnten innerhalb von 10 Sekunden angehalten werden')
                    time.sleep(0.01)
            yield
        finally:
            if resume:
                try:
                    identity.seek(0)
                    current = identity.read().rsplit(')', 1)[1].split()[19]
                    if current == original:
                        os.kill(pid, signal.SIGCONT)
                except (ProcessLookupError, FileNotFoundError):
                    pass


def parse_mappings(text, anonymous):
    mappings = []
    for line in text.splitlines():
        fields = line.split(None, 5)
        start, end = (int(n, 16) for n in fields[0].split('-'))
        reason = None
        if start >= (1 << 63):
            reason = 'kernel_address'
        elif 'r' not in fields[1]:
            reason = 'not_readable'
        elif anonymous and fields[4] != '0':
            reason = 'file_backed'
        mappings.append({
            'start': hex(start), 'end': hex(end), 'permissions': fields[1],
            'file_offset': fields[2], 'device': fields[3], 'inode': fields[4],
            'pathname': fields[5] if len(fields) == 6 else '',
            'file': None if reason else 'regions/{:016x}-{:016x}.bin'.format(start, end),
            'status': 'skipped' if reason else 'pending', 'skip_reason': reason,
            'unreadable_ranges': [], 'bytes_read': 0, 'bytes_zero': 0, 'captured_until': 0,
        })
    return mappings


def elf_info(source):
    """Read ELF section metadata without binutils or third-party dependencies."""
    size = os.fstat(source.fileno()).st_size

    def at(offset, length):
        if offset < 0 or length < 0 or offset + length > size:
            raise ValueError('ELF-Bereich außerhalb der Datei')
        data = os.pread(source.fileno(), length, offset)
        if len(data) != length:
            raise ValueError('Verkürzte ELF-Datei')
        return data

    ident = at(0, min(16, size))
    if len(ident) < 16 or ident[:4] != b'\x7fELF':
        return None
    if ident[4] not in (1, 2) or ident[5] not in (1, 2):
        raise ValueError('Unbekanntes ELF-Format')
    order = '<' if ident[5] == 1 else '>'
    header = struct.Struct(order + ('HHIIIIIHHHHHH' if ident[4] == 1 else 'HHIQQQIHHHHHH'))
    section = struct.Struct(order + ('IIIIIIIIII' if ident[4] == 1 else 'IIQQQQIIQQ'))
    values = header.unpack(at(16, header.size))
    offset, stride, count, names_index = values[5], values[10], values[11], values[12]
    result = {'build_id': None, 'debuglink': None, 'debugaltlink': None,
              'embedded_debug': False, 'elf_class': 32 if ident[4] == 1 else 64}
    if not offset:
        return result
    if stride < section.size:
        raise ValueError('Ungültige ELF-Section-Größe')
    first = section.unpack(at(offset, section.size))
    count = count or first[5]
    names_index = first[6] if names_index == 0xffff else names_index
    if count > 1000000 or names_index >= count or offset + stride * count > size:
        raise ValueError('Ungültige ELF-Section-Tabelle')
    names_header = section.unpack(at(offset + stride * names_index, section.size))
    if names_header[5] > 16 * CHUNK:
        raise ValueError('Zu große ELF-Section-Namentabelle')
    names = at(names_header[4], names_header[5])
    for index in range(count):
        entry = section.unpack(at(offset + stride * index, section.size))
        name = names[entry[0]:].split(b'\0', 1)[0]
        if name in (b'.debug_info', b'.zdebug_info'):
            result['embedded_debug'] = True
        if entry[1] != 7 and name not in (b'.gnu_debuglink', b'.gnu_debugaltlink'):
            continue
        if entry[5] > CHUNK:
            continue
        data = at(entry[4], entry[5])
        if name in (b'.gnu_debuglink', b'.gnu_debugaltlink'):
            filename, separator, remainder = data.partition(b'\0')
            if not separator:
                continue
            filename = os.fsdecode(filename)
            if name == b'.gnu_debuglink':
                crc_offset = (len(os.fsencode(filename)) + 4) & ~3
                if crc_offset + 4 <= len(data) and os.path.basename(filename) == filename:
                    result['debuglink'] = {'name': filename, 'crc32': struct.unpack_from(order + 'I', data, crc_offset)[0]}
            else:
                result['debugaltlink'] = {'name': filename, 'build_id': remainder.hex()}
        elif entry[1] == 7:
            position = 0
            while position + 12 <= len(data):
                namesz, descsz, kind = struct.unpack_from(order + 'III', data, position)
                position += 12
                desc = position + ((namesz + 3) & ~3)
                finish = desc + descsz
                if finish > len(data):
                    break
                if kind == 3 and data[position:position + namesz].rstrip(b'\0') == b'GNU':
                    result['build_id'] = data[desc:finish].hex()
                position = desc + ((descsz + 3) & ~3)
    return result


def file_crc32(source):
    source.seek(0)
    crc = 0
    while True:
        data = source.read(CHUNK)
        if not data:
            return crc
        crc = zlib.crc32(data, crc)


def symbol_candidates(path, info, directories):
    """GDB-style build ID/debuglink lookup, plus DWZ supplementary files."""
    build_id = info['build_id']
    if build_id and len(build_id) > 2:
        for directory in directories:
            yield (os.path.join(directory, '.build-id', build_id[:2], build_id[2:] + '.debug'),
                   'build_id', build_id)
    link = info['debuglink']
    if link:
        parent = os.path.dirname(path)
        for directory in (parent, os.path.join(parent, '.debug')):
            yield os.path.join(directory, link['name']), 'crc32', link['crc32']
        for directory in directories:
            yield os.path.join(directory, parent.lstrip('/'), link['name']), 'crc32', link['crc32']
    link = info['debugaltlink']
    if link:
        yield os.path.normpath(os.path.join(os.path.dirname(path), link['name'])), 'build_id', link['build_id']
        for directory in directories:
            yield os.path.normpath(os.path.join(directory, link['name'].lstrip('/'))), 'build_id', link['build_id']
            if len(link['build_id']) > 2:
                yield (os.path.join(directory, '.build-id', link['build_id'][:2], link['build_id'][2:] + '.debug'),
                       'build_id', link['build_id'])


def capture_artifacts(pid, mappings, executable_path, debug_dirs, saved_artifacts=()):
    """Stream the actual mapped ELF files and matching installed symbols."""
    directories = list(dict.fromkeys(['/usr/lib/debug'] + list(debug_dirs)))
    roots = ['/proc/{}/root'.format(pid), '']
    seen = {}
    saved_by_name = {item['file']: item for item in saved_artifacts if item['status'] == 'complete'}

    def archive(source, path, role, info, parent=None):
        st = os.fstat(source.fileno())
        key = (st.st_dev, st.st_ino)
        if key in seen:
            yield b'L', json_bytes({'parent': parent, 'file': seen[key], 'source_path': path})
            return
        filename = 'executable' if role == 'executable' else '{}/{}-{}'.format(
            'debug' if role == 'debug' else 'binaries',
            hashlib.sha256(os.fsencode(path)).hexdigest()[:16],
            os.path.basename(path) or 'unnamed')
        seen[key] = filename
        details = {'file': filename, 'source_path': path, 'role': role,
                   'parent': parent, 'size': st.st_size, 'elf': info, 'status': 'partial'}
        saved = saved_by_name.get(filename)
        reuse = saved and saved['size'] == st.st_size and sha256_file(source) == saved['sha256']
        if reuse:
            details.update(status='complete', sha256=saved['sha256'], bytes_saved=st.st_size)
            yield b'V', json_bytes(details)
        else:
            yield b'A', json_bytes(details)
        source.seek(0)
        digest = hashlib.sha256()
        count = 0
        while not reuse:
            block = source.read(CHUNK)
            if not block:
                break
            digest.update(block)
            count += len(block)
            yield b'F', block
        if not reuse and count != st.st_size:
            raise RuntimeError('Dateigröße während der Sicherung geändert: {}'.format(path))
        if not reuse:
            yield b'C', json_bytes({'sha256': digest.hexdigest(), 'bytes_saved': count})
        if info is None:
            return
        matched = set()
        for candidate, check, expected in symbol_candidates(path, info, directories):
            category = 'supplementary' if info['debugaltlink'] and expected == info['debugaltlink']['build_id'] else 'separate'
            if role == 'debug' and category != 'supplementary':
                continue
            if category in matched:
                continue
            for root in roots:
                try:
                    debug = open(root + '/' + candidate.lstrip('/'), 'rb')
                except (FileNotFoundError, NotADirectoryError):
                    continue
                except OSError as exc:
                    yield b'W', json_bytes('Debug-Datei {}: {}'.format(candidate, exc))
                    continue
                with debug:
                    try:
                        debug_info = elf_info(debug)
                        valid = debug_info is not None and (
                            debug_info['build_id'] == expected if check == 'build_id' else file_crc32(debug) == expected)
                    except (ValueError, OSError) as exc:
                        yield b'W', json_bytes('Debug-Datei {}: {}'.format(candidate, exc))
                        continue
                    if not valid:
                        yield b'W', json_bytes('Debug-Datei passt nicht zu {}: {}'.format(path, candidate))
                        continue
                    matched.add(category)
                    yield from archive(debug, candidate, 'debug', debug_info, filename)
                    break
        if not info['embedded_debug'] and 'separate' not in matched and role != 'debug':
            yield b'W', json_bytes('Keine passenden installierten DWARF-Debug-Symbole gefunden: {}'.format(path))
        if info['debugaltlink'] and 'supplementary' not in matched:
            yield b'W', json_bytes('Ergänzende DWZ-Debug-Datei fehlt: {}'.format(info['debugaltlink']['name']))

    try:
        exe = open('/proc/{}/exe'.format(pid), 'rb')
    except OSError as exc:
        yield b'W', json_bytes('Executable konnte nicht gesichert werden: {}'.format(exc))
    else:
        with exe:
            try:
                info = elf_info(exe)
            except (ValueError, OSError) as exc:
                info = None
                yield b'W', json_bytes('ELF-Metadaten des Executable: {}'.format(exc))
            yield from archive(exe, executable_path or '/unknown-executable', 'executable', info)
    processed = set()
    for region in mappings:
        path = region['pathname']
        identity = (region['device'], region['inode'])
        if not path.startswith('/') or region['inode'] == '0' or identity in processed:
            continue
        processed.add(identity)
        mapped = '/proc/{}/map_files/{:x}-{:x}'.format(pid, int(region['start'], 16), int(region['end'], 16))
        paths = [mapped]
        if not path.endswith(' (deleted)'):
            paths.append('/proc/{}/root{}'.format(pid, path))
        source = None
        failures = []
        for candidate in paths:
            try:
                source = open(candidate, 'rb')
                st = os.fstat(source.fileno())
                major, minor = (int(n, 16) for n in region['device'].split(':'))
                if st.st_ino != int(region['inode']) or (os.major(st.st_dev), os.minor(st.st_dev)) != (major, minor):
                    source.close()
                    source = None
                    failures.append('Dateiidentität geändert')
                    continue
                break
            except OSError as exc:
                failures.append(str(exc))
        if source is None:
            yield b'W', json_bytes('Mapping-Datei {} nicht gesichert: {}'.format(path, '; '.join(failures)))
            continue
        with source:
            try:
                info = elf_info(source)
            except (ValueError, OSError) as exc:
                yield b'W', json_bytes('ELF-Metadaten {}: {}'.format(path, exc))
                continue
            if info is not None:
                yield from archive(source, path, 'library', info)


def capture_events(pid, anonymous=False, stop=False, debug_dirs=(), resume=None):
    """Bounded event stream shared by local capture and the SSH producer."""
    page = os.sysconf('SC_PAGE_SIZE')
    with open('/proc/{}/mem'.format(pid), 'rb', buffering=0) as mem:
        with stopped_process(pid, stop):
            files = {}
            warnings = []
            for name in ('maps', 'smaps', 'status', 'auxv', 'cmdline'):
                try:
                    with open('/proc/{}/{}'.format(pid, name), 'rb') as source:
                        files[name] = base64.b64encode(source.read()).decode('ascii')
                except OSError as exc:
                    if name == 'maps':
                        raise
                    warnings.append('{}: {}'.format(name, exc))
            maps = base64.b64decode(files['maps']).decode('utf-8', errors='replace')
            try:
                executable_path = os.readlink('/proc/{}/exe'.format(pid))
            except OSError as exc:
                executable_path = None
                warnings.append('exe: {}'.format(exc))
            metadata = {
                'format': 'ramdump-regions', 'version': 4, 'pid': pid,
                'hostname': platform.node(), 'machine': platform.machine(),
                'byte_order': sys.byteorder, 'page_size': page,
                'started_at': now(), 'finished_at': None,
                'capture_mode': 'stopped' if stop else 'live',
                'anonymous_only': anonymous, 'executable_path': executable_path,
                'executable_saved': False, 'complete': False,
                'warnings': warnings, 'mappings': parse_mappings(maps, anonymous),
                'artifacts': [], 'artifact_links': [],
                'debug_directories': list(dict.fromkeys(['/usr/lib/debug'] + list(debug_dirs))),
                'bytes_read': 0, 'bytes_zero': 0, 'bytes_unreadable': 0,
                'process_identity': process_identity(pid),
            }
            if resume:
                resume_regions(metadata, resume)
            yield b'M', json_bytes({'manifest': metadata, 'files': files})
            yield from capture_artifacts(pid, metadata['mappings'], executable_path, debug_dirs,
                                         resume.get('artifacts', ()) if resume else ())
            refresh_blocks = {region_key(r): r.get('refresh_blocks', {}) for r in resume['mappings']} if resume and resume.get('refresh') else {}
            for index, region in enumerate(metadata['mappings']):
                if region['file'] is None or region['status'] == 'complete':
                    continue
                yield b'R', json_bytes(index)
                address = int(region['start'], 16) + region['captured_until']
                end = int(region['end'], 16)
                retry_until = address
                while address < end:
                    size = min(page if address < retry_until else CHUNK, end - address)
                    try:
                        data = os.pread(mem.fileno(), size, address)
                    except OSError as exc:
                        if exc.errno not in READ_ERRORS:
                            raise
                        if size > page:
                            retry_until = address + size
                            continue
                        yield b'H', json_bytes({'start': hex(address), 'end': hex(address + size), 'errno': exc.errno})
                        address += size
                        continue
                    if not data:
                        raise RuntimeError('Zielprozess beendet oder Speicher nicht mehr verfügbar')
                    offset = address - int(region['start'], 16)
                    saved = refresh_blocks.get(region_key(region), {}).get(str(offset))
                    if saved and saved['size'] == len(data) and saved['sha256'] == hashlib.sha256(data).hexdigest():
                        yield b'K', json_bytes(dict(saved, start=hex(address), end=hex(address + len(data))))
                    else:
                        yield from memory_events(address, data, page)
                    address += len(data)
                yield b'E', b''
    # The process has resumed before completion is reported.
    yield b'Z', json_bytes({'finished_at': now()})


def send_stream(events, compress=True):
    out = sys.stdout.buffer
    out.write(MAGIC)
    try:
        for kind, payload in events:
            if compress and len(payload) >= 256:
                packed = zlib.compress(payload, 1)
                if len(packed) + HEADER.size < len(payload):
                    payload = HEADER.pack(kind, len(payload)) + packed
                    kind = b'z'
            out.write(HEADER.pack(kind, len(payload)))
            out.write(payload)
    except (OSError, RuntimeError, ValueError, zlib.error) as exc:
        if isinstance(exc, BrokenPipeError):
            raise
        payload = json_bytes(str(exc))
        out.write(HEADER.pack(b'!', len(payload)))
        out.write(payload)
    out.flush()


def read_exact(stream, size):
    data = bytearray()
    while len(data) < size:
        block = stream.read(size - len(data))
        if not block:
            raise RuntimeError('SSH-Datenstrom vorzeitig beendet; Dump ist unvollständig')
        data.extend(block)
    return bytes(data)


def receive_stream(stream, transferred=None):
    if read_exact(stream, len(MAGIC)) != MAGIC:
        raise RuntimeError('Ungültiger SSH-Datenstrom (zusätzliche Shell-Ausgaben?)')
    while True:
        kind, size = HEADER.unpack(read_exact(stream, HEADER.size))
        if size > (256 * CHUNK if kind in (b'M', b'z') else CHUNK + ADDRESS.size):
            raise RuntimeError('Ungültige Datensatzgröße im SSH-Datenstrom')
        payload = read_exact(stream, size)
        if transferred:
            transferred(HEADER.size + size)
        if kind == b'z':
            if len(payload) < HEADER.size:
                raise RuntimeError('Verkürzter komprimierter Datensatz')
            kind, original_size = HEADER.unpack(payload[:HEADER.size])
            limit = 256 * CHUNK if kind == b'M' else CHUNK + ADDRESS.size
            if kind == b'z' or original_size > limit:
                raise RuntimeError('Ungültige Größe komprimierter Daten')
            decoder = zlib.decompressobj()
            unpacked = decoder.decompress(payload[HEADER.size:], original_size + 1)
            if len(unpacked) != original_size or not decoder.eof or decoder.unused_data or decoder.unconsumed_tail:
                raise RuntimeError('Ungültiger komprimierter Datensatz')
            payload = unpacked
        if kind == b'!':
            raise RuntimeError(json.loads(payload))
        yield kind, payload
        if kind == b'Z':
            return


def write_sparse(fd, offset, data):
    page = 4096
    zero = bytes(page)
    for position in range(0, len(data), page):
        block = data[position:position + page]
        if block == zero[:len(block)]:
            continue
        target = offset + position
        while block:
            count = os.pwrite(fd, block, target)
            if count == 0:
                raise OSError('Schreiben lieferte 0 Bytes')
            target += count
            block = block[count:]


def zero_range(fd, offset, count):
    """Clear old bytes on refresh, retaining sparse holes where supported."""
    count = min(count, max(0, os.fstat(fd).st_size - offset))
    if not count:
        return
    if PUNCH_HOLE is not None:
        if PUNCH_HOLE(fd, 3, offset, count) == 0:  # KEEP_SIZE | PUNCH_HOLE
            return
        error = ctypes.get_errno()
        if error not in (errno.EINVAL, errno.EOPNOTSUPP, errno.ENOSYS):
            raise OSError(error, os.strerror(error))
    zero = bytes(min(CHUNK, count))
    while count:
        written = os.pwrite(fd, zero[:min(count, len(zero))], offset)
        if not written:
            raise OSError('Schreiben lieferte 0 Bytes')
        offset += written
        count -= written


class DumpWriter:
    def __init__(self, output, ssh, continuing=False, refresh=False):
        self.output = Path(output)
        self.ssh = ssh
        self.refresh = refresh
        self.continuing = continuing or refresh
        self.resume_request = None
        self.refresh_regions = {}
        self.manifest = {'format': 'ramdump-regions', 'version': 4, 'complete': False}
        self.region = None
        self.fd = None
        self.artifact = None
        self.artifact_file = None
        self.artifact_digest = None
        self.artifact_bytes = 0
        self.finished = False
        self.last_checkpoint = 0
        self.active_metadata = not self.continuing
        self.total = 0
        self.done = 0
        self.zero = 0
        self.wire_bytes = 0
        self.phase = 'Verbindung / Pruefung'
        if not self.continuing:
            self.output.mkdir(mode=0o700)
        self.lock_fd = os.open(self.output / '.lock', os.O_WRONLY | os.O_CREAT | os.O_NOFOLLOW, 0o600)
        try:
            try:
                fcntl.flock(self.lock_fd, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError:
                raise RuntimeError('Ein anderer Dump-Prozess verwendet diesen Ordner') from None
            if self.continuing:
                self.prepare_resume()
            for directory in ('regions', 'binaries', 'debug'):
                (self.output / directory).mkdir(mode=0o700, exist_ok=True)
            if not self.continuing:
                self.checkpoint(force=True)
        except BaseException:
            os.close(self.lock_fd)
            raise

    def prepare_resume(self):
        with (self.output / 'manifest.json').open(encoding='utf-8') as source:
            previous = json.load(source)
        if previous.get('format') != 'ramdump-regions' or previous.get('version') not in (2, 3, 4):
            raise RuntimeError('Unbekanntes Dump-Format')
        if previous.get('complete') and not self.refresh:
            raise RuntimeError('Dieser Dump ist bereits vollständig')
        if previous.get('ssh') != self.ssh:
            raise RuntimeError('Zum Fortsetzen dasselbe --ssh-Ziel verwenden')
        for region in previous['mappings']:
            length = int(region['end'], 16) - int(region['start'], 16)
            region.setdefault('bytes_zero', 0)
            if previous['version'] < 4:
                region['captured_until'] = length if region['status'] == 'complete' else 0
                if region['status'] != 'complete':
                    region.update(bytes_read=0, unreadable_ranges=[])
            cursor = region.setdefault('captured_until', 0)
            if not 0 <= cursor <= length:
                raise RuntimeError('Ungültiger Zwischenstand im Manifest')
            if region.get('file'):
                expected = region_file(region)
                if cursor:
                    try:
                        saved = (self.output / expected).stat(follow_symlinks=False)
                    except FileNotFoundError:
                        saved = None
                    valid = saved is not None and stat.S_ISREG(saved.st_mode) and saved.st_size >= cursor
                    valid = valid and (region['status'] != 'complete' or saved.st_size == length)
                    if not valid:
                        if not self.refresh:
                            raise RuntimeError('Gespeichertes Mapping fehlt oder hat eine falsche Größe: ' + expected)
                        region.update(status='pending', captured_until=0, bytes_read=0, bytes_zero=0, unreadable_ranges=[])
        artifacts = previous.setdefault('artifacts', [])
        if not previous.get('process_identity'):
            with (self.output / 'executable').open('rb') as source:
                digest = sha256_file(source)
                size = os.fstat(source.fileno()).st_size
            previous['legacy_executable_sha256'] = digest
            if not any(item['file'] == 'executable' for item in artifacts):
                artifacts.append({'file': 'executable', 'role': 'executable', 'status': 'complete',
                                  'size': size, 'bytes_saved': size, 'sha256': digest})
        for item in artifacts:
            name = self.artifact_name(item['file'])
            if item['status'] != 'complete':
                continue
            try:
                with (self.output / name).open('rb') as source:
                    if os.fstat(source.fileno()).st_size != item['size'] or sha256_file(source) != item['sha256']:
                        item['status'] = 'partial'
            except FileNotFoundError:
                item['status'] = 'partial'
        self.manifest = previous
        previous['refresh'] = self.refresh
        self.resume_request = previous

    def prepare_refresh(self):
        """Hash only trustworthy local prefixes; unknown pages are always retried."""
        self.phase = 'Lokale SHA256-Pruefung'
        self.total = sum(r.get('captured_until', 0) for r in self.resume_request['mappings'] if r['file'])
        self.done = 0
        self.refresh_regions = {region_key(r): r for r in self.resume_request['mappings']}
        for region in self.resume_request['mappings']:
            region.pop('refresh_blocks', None)
            cursor = region.get('captured_until', 0)
            if not region['file'] or not cursor:
                continue
            blocks = {}
            holes = region['unreadable_ranges']
            start = int(region['start'], 16)
            with (self.output / region_file(region)).open('rb') as source:
                for offset in range(0, cursor, CHUNK):
                    data = source.read(min(CHUNK, cursor - offset))
                    if len(data) != min(CHUNK, cursor - offset):
                        raise RuntimeError('Lokales Mapping während der Prüfung verkürzt')
                    if not any(int(h['start'], 16) < start + offset + len(data) and int(h['end'], 16) > start + offset for h in holes):
                        zero = bytes(4096)
                        zero_bytes = sum(len(data[p:p + 4096]) for p in range(0, len(data), 4096)
                                         if data[p:p + 4096] == zero[:min(4096, len(data) - p)])
                        blocks[str(offset)] = {'size': len(data), 'sha256': hashlib.sha256(data).hexdigest(),
                                               'bytes_zero': zero_bytes}
                    self.done += len(data)
            region['refresh_blocks'] = blocks

    @staticmethod
    def artifact_name(filename):
        name = Path(filename)
        if name != Path('executable') and not (
                len(name.parts) == 2 and name.parts[0] in ('binaries', 'debug')
                and name.parts[1] not in ('.', '..')):
            raise RuntimeError('Ungültiger Zusatzdatei-Name')
        return name

    def checkpoint(self, force=False):
        if not force and time.monotonic() - self.last_checkpoint < 5:
            return
        if self.fd is not None:
            cursor = self.region['captured_until']
            if not self.refresh or os.fstat(self.fd).st_size < cursor:
                os.ftruncate(self.fd, cursor)
            os.fsync(self.fd)
        # Persist file names before publishing a manifest that references them.
        for path in (self.output / 'regions', self.output / 'binaries', self.output / 'debug'):
            directory = os.open(path, os.O_RDONLY | os.O_DIRECTORY)
            try:
                os.fsync(directory)
            finally:
                os.close(directory)
        temporary = self.output / 'manifest.json.tmp'
        with temporary.open('w', encoding='utf-8') as target:
            json.dump(self.manifest, target, ensure_ascii=True, indent=2)
            target.write('\n')
            target.flush()
            os.fsync(target.fileno())
        temporary.replace(self.output / 'manifest.json')
        directory = os.open(self.output, os.O_RDONLY | os.O_DIRECTORY)
        try:
            os.fsync(directory)
        finally:
            os.close(directory)
        self.last_checkpoint = time.monotonic()

    def transferred(self, count):
        self.wire_bytes += count

    def advance(self, address, count):
        start = int(self.region['start'], 16)
        if (count <= 0 or address != start + self.region['captured_until']
                or address + count > int(self.region['end'], 16)):
            raise RuntimeError('Nicht fortlaufende Daten oder Bereich außerhalb der Region')
        self.region['captured_until'] += count
        self.done += count

    def accept(self, kind, payload):
        if kind == b'M':
            snapshot = json.loads(payload)
            self.manifest = snapshot['manifest']
            self.manifest['ssh'] = self.ssh
            self.active_metadata = True
            self.total = sum(int(r['end'], 16) - int(r['start'], 16) for r in self.manifest['mappings'] if r['file'])
            self.done = sum(r.get('captured_until', 0) for r in self.manifest['mappings'])
            self.zero = self.manifest.get('bytes_zero', 0)
            self.phase = 'Binaries / Debug-Symbole'
            for name, data in snapshot['files'].items():
                if name not in ('maps', 'smaps', 'status', 'auxv', 'cmdline'):
                    raise RuntimeError('Ungültiger Metadaten-Dateiname')
                suffix = '.bin' if name in ('auxv', 'cmdline') else '.txt'
                (self.output / (name + suffix)).write_bytes(base64.b64decode(data))
            self.checkpoint(force=True)
        elif kind == b'W':
            self.manifest['warnings'].append(json.loads(payload))
        elif kind == b'A':
            if self.artifact_file is not None:
                raise RuntimeError('Vorherige Zusatzdatei nicht abgeschlossen')
            self.artifact = json.loads(payload)
            name = self.artifact_name(self.artifact['file'])
            flags = os.O_WRONLY | os.O_CREAT | os.O_NOFOLLOW | (os.O_TRUNC if self.continuing else os.O_EXCL)
            self.artifact_file = os.fdopen(os.open(self.output / name, flags, 0o600), 'wb')
            self.artifact_digest = hashlib.sha256()
            self.artifact_bytes = 0
            self.manifest['artifacts'].append(self.artifact)
            self.checkpoint()
        elif kind == b'F':
            self.artifact_file.write(payload)
            self.artifact_digest.update(payload)
            self.artifact_bytes += len(payload)
        elif kind == b'C':
            details = json.loads(payload)
            if (details['sha256'] != self.artifact_digest.hexdigest()
                    or details['bytes_saved'] != self.artifact_bytes
                    or self.artifact['size'] != self.artifact_bytes):
                raise RuntimeError('Prüfsumme oder Größe der Zusatzdatei stimmt nicht')
            self.artifact_file.flush()
            os.fsync(self.artifact_file.fileno())
            self.artifact_file.close()
            self.artifact_file = None
            self.artifact.update(details, status='complete')
            if self.artifact['role'] == 'executable':
                self.manifest['executable_saved'] = True
            self.artifact = None
            self.checkpoint()
        elif kind == b'V':
            item = json.loads(payload)
            self.artifact_name(item['file'])
            saved = next((a for a in self.resume_request['artifacts'] if a['file'] == item['file']), None) if self.resume_request else None
            if not saved or saved['status'] != 'complete' or saved['sha256'] != item['sha256']:
                raise RuntimeError('Zusatzdatei wurde nicht zum Wiederverwenden freigegeben')
            self.manifest['artifacts'].append(item)
            if item['role'] == 'executable':
                self.manifest['executable_saved'] = True
        elif kind == b'L':
            self.manifest['artifact_links'].append(json.loads(payload))
            self.checkpoint()
        elif kind == b'R':
            if self.fd is not None:
                raise RuntimeError('Vorherige Region nicht abgeschlossen')
            self.region = self.manifest['mappings'][json.loads(payload)]
            start = int(self.region['start'], 16)
            end = int(self.region['end'], 16)
            expected = region_file(self.region)
            if end <= start:
                raise RuntimeError('Ungültige Mapping-Datei')
            flags = os.O_WRONLY | os.O_CREAT | os.O_NOFOLLOW | (0 if self.continuing else os.O_EXCL)
            self.fd = os.open(self.output / expected, flags, 0o600)
            if not self.refresh:
                os.ftruncate(self.fd, self.region['captured_until'])
            self.region['status'] = 'partial'
            self.phase = 'Region ' + self.region['start']
            self.checkpoint()
        elif kind == b'D':
            address = ADDRESS.unpack(payload[:ADDRESS.size])[0]
            data = payload[ADDRESS.size:]
            start = int(self.region['start'], 16)
            if not data or address != start + self.region['captured_until'] or address + len(data) > int(self.region['end'], 16):
                raise RuntimeError('Daten außerhalb der aktuellen Region')
            write_sparse(self.fd, address - start, data)
            self.advance(address, len(data))
            self.region['bytes_read'] += len(data)
            self.manifest['bytes_read'] += len(data)
            if self.refresh:
                self.manifest['bytes_refreshed'] += len(data)
            self.checkpoint()
        elif kind == b'N':
            address, count = RANGE.unpack(payload)
            start = int(self.region['start'], 16)
            if address != start + self.region['captured_until'] or count <= 0 or address + count > int(self.region['end'], 16):
                raise RuntimeError('Ungültiger Nullbereich')
            zero_range(self.fd, address - start, count)
            self.advance(address, count)
            self.region['bytes_read'] += count
            self.region['bytes_zero'] += count
            self.manifest['bytes_read'] += count
            self.manifest['bytes_zero'] += count
            self.zero += count
            if self.refresh:
                self.manifest['bytes_refreshed'] += count
            self.checkpoint()
        elif kind == b'K':
            block = json.loads(payload)
            address = int(block['start'], 16)
            offset = address - int(self.region['start'], 16)
            previous = self.refresh_regions.get(region_key(self.region)) if self.refresh else None
            saved = previous.get('refresh_blocks', {}).get(str(offset)) if previous else None
            if not saved or any(block[key] != saved[key] for key in ('size', 'sha256', 'bytes_zero')):
                raise RuntimeError('Nicht freigegebener SHA256-Block')
            self.advance(address, block['size'])
            self.region['bytes_read'] += block['size']
            self.region['bytes_zero'] += block['bytes_zero']
            self.manifest['bytes_read'] += block['size']
            self.manifest['bytes_zero'] += block['bytes_zero']
            self.manifest['bytes_reused'] += block['size']
            self.zero += block['bytes_zero']
            self.checkpoint()
        elif kind == b'H':
            hole = json.loads(payload)
            count = int(hole['end'], 16) - int(hole['start'], 16)
            start = int(self.region['start'], 16)
            if int(hole['start'], 16) != start + self.region['captured_until'] or count <= 0 or int(hole['end'], 16) > int(self.region['end'], 16):
                raise RuntimeError('Ungültiger unlesbarer Bereich')
            zero_range(self.fd, int(hole['start'], 16) - start, count)
            self.advance(int(hole['start'], 16), count)
            holes = self.region['unreadable_ranges']
            if holes and holes[-1]['end'] == hole['start'] and holes[-1]['errno'] == hole['errno']:
                holes[-1]['end'] = hole['end']
            else:
                holes.append(hole)
            self.manifest['bytes_unreadable'] += count
            self.checkpoint()
        elif kind == b'E':
            if self.region['captured_until'] != int(self.region['end'], 16) - int(self.region['start'], 16):
                raise RuntimeError('Unvollständige Region')
            os.ftruncate(self.fd, int(self.region['end'], 16) - int(self.region['start'], 16))
            os.fsync(self.fd)
            os.close(self.fd)
            self.fd = None
            self.region['status'] = 'complete'
            self.region = None
            self.checkpoint()
        elif kind == b'Z':
            if self.fd is not None or self.artifact_file is not None:
                raise RuntimeError('Dump-Ende innerhalb einer Datei')
            self.manifest.update(json.loads(payload))
            self.finished = True
            self.phase = 'Abschluss'
        else:
            raise RuntimeError('Unbekannter Datensatz: {!r}'.format(kind))

    def close(self):
        try:
            if self.active_metadata:
                self.checkpoint(force=True)
        finally:
            if self.fd is not None:
                os.close(self.fd)
                self.fd = None
            if self.artifact_file is not None:
                self.artifact_file.close()
                self.artifact_file = None
            os.close(self.lock_fd)


class Progress:
    def __init__(self, writer, enabled=True):
        self.writer = writer
        self.enabled = enabled
        self.stopped = threading.Event()
        self.tty = sys.stderr.isatty()
        self.thread = None
        self.last_done = None
        self.last_total = None
        self.last_time = time.monotonic()

    def render(self):
        writer = self.writer
        done, total = writer.done, writer.total
        instant = time.monotonic()
        rate = 0 if self.last_done is None or total != self.last_total else max(0, done - self.last_done) / max(instant - self.last_time, 0.001)
        self.last_done, self.last_time = done, instant
        self.last_total = total
        fraction = min(done / total, 1) if total else 0
        filled = int(fraction * 20)
        bar = '#' * filled + '-' * (20 - filled)
        eta = '{:.0f}s'.format((total - done) / rate) if rate else '--'
        text = '[{}] {:5.1f}% {:.2f}/{:.2f} GiB | {:.1f} MiB/s | ETA {}'.format(
            bar, fraction * 100, done / (1 << 30), total / (1 << 30), rate / (1 << 20), eta)
        text += ' | SSH {:.2f} GiB | Null {:.2f} GiB | {}'.format(
            writer.wire_bytes / (1 << 30), writer.zero / (1 << 30), writer.phase)
        if self.tty:
            width = shutil.get_terminal_size((120, 24)).columns
            sys.stderr.write('\r' + text[:max(1, width - 1)].ljust(max(1, width - 1)))
        else:
            sys.stderr.write(text + '\n')
        sys.stderr.flush()

    def run(self):
        while not self.stopped.is_set():
            self.render()
            self.stopped.wait(0.5 if self.tty else 5)

    def start(self):
        if self.enabled:
            self.thread = threading.Thread(target=self.run, daemon=True)
            self.thread.start()

    def close(self):
        if self.thread is not None:
            self.stopped.set()
            self.thread.join()
            self.render()
            if self.tty:
                sys.stderr.write('\n')


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('pid', type=int, help='PID auf dem Zielrechner')
    parser.add_argument('-o', '--output', help='Lokaler Zielordner (Standard: dump-PID/)')
    parser.add_argument('--ssh', metavar='USER@HOST', help='Remote lesen; alle Dateien lokal aufbauen')
    parser.add_argument('--anonymous', action='store_true', help='Nur Mappings ohne Datei-Inode (für Go-Analyse möglichst weglassen)')
    parser.add_argument('--sudo', action='store_true', help='Remote python3 über sudo -n starten (mit --ssh)')
    parser.add_argument('--stop', action='store_true', help='Ziel während der Aufnahme mit SIGSTOP pausieren, danach fortsetzen')
    parser.add_argument('--debug-dir', action='append', default=[], metavar='PATH',
                        help='Zusätzliches Debug-Symbolverzeichnis auf dem Zielrechner (wiederholbar)')
    parser.add_argument('-c', '--continue', dest='continuing', action='store_true',
                        help='Vorhandenen Teildump desselben Prozesses fortsetzen')
    parser.add_argument('--refresh', action='store_true',
                        help='Impliziert -c: Speicherblöcke per SHA256 mit dem Prozess vergleichen und aktualisieren')
    parser.add_argument('--no-compress', action='store_true', help='SSH-Daten nicht komprimieren')
    parser.add_argument('--no-progress', action='store_true', help='Fortschrittsanzeige ausschalten')
    parser.add_argument('--stream', action='store_true', help=argparse.SUPPRESS)
    parser.add_argument('--resume-input', action='store_true', help=argparse.SUPPRESS)
    args = parser.parse_args()
    if args.pid <= 0:
        parser.error('PID muss positiv sein')
    if args.sudo and not args.ssh:
        parser.error('--sudo benötigt --ssh')
    if args.ssh and (args.ssh.startswith('-') or not args.ssh.strip()):
        parser.error('Ungültiges SSH-Ziel')
    if args.stream:
        resume = json.load(sys.stdin) if args.resume_input else None
        events = capture_events(args.pid, args.anonymous, args.stop, args.debug_dir, resume)
        try:
            send_stream(events, compress=not args.no_compress)
        finally:
            events.close()
        return

    output = args.output or 'dump-{}'.format(args.pid)
    writer = DumpWriter(output, args.ssh, args.continuing, args.refresh)
    if args.refresh:
        print('Refresh vergleicht den vorhandenen Dump mit dem aktuellen Prozess und aktualisiert geänderte Blöcke.', file=sys.stderr)
    if args.continuing:
        print('Fortsetzen kombiniert Daten mehrerer Aufnahmezeitpunkte; es entsteht kein einheitlicher Snapshot.', file=sys.stderr)
        if writer.resume_request['version'] < 4:
            print('Alter Dump: fertige Regionen bleiben erhalten; angebrochene Regionen werden neu gelesen.', file=sys.stderr)
    progress = Progress(writer, not args.no_progress)
    child = None
    events = None
    progress.start()
    try:
        if args.refresh:
            writer.prepare_refresh()
        if args.ssh:
            with open(__file__, encoding='utf-8') as script:
                source = script.read()
            command = (['sudo', '-n'] if args.sudo else []) + ['python3', '-c', source, str(args.pid), '--stream']
            for option in ('anonymous', 'stop', 'no_compress'):
                if getattr(args, option):
                    command.append('--' + option.replace('_', '-'))
            for directory in args.debug_dir:
                command.extend(['--debug-dir', directory])
            if writer.resume_request:
                command.append('--resume-input')
            child = subprocess.Popen(['ssh', '-T', args.ssh, ' '.join(shlex.quote(arg) for arg in command)],
                                     stdin=subprocess.PIPE if writer.resume_request else subprocess.DEVNULL,
                                     stdout=subprocess.PIPE)
            if writer.resume_request:
                child.stdin.write(json_bytes(writer.resume_request))
                child.stdin.close()
            events = receive_stream(child.stdout, writer.transferred)
        else:
            events = capture_events(args.pid, args.anonymous, args.stop, args.debug_dir, writer.resume_request)
        for kind, payload in events:
            writer.accept(kind, payload)
        if child is not None:
            child.stdout.close()
            if child.wait() != 0:
                raise RuntimeError('SSH/Remote-Skript meldet einen Fehler')
        if not writer.finished:
            raise RuntimeError('Dump ist unvollständig')
        writer.manifest['complete'] = True
    except BaseException as exc:
        writer.manifest['error'] = str(exc) or type(exc).__name__
        raise
    finally:
        try:
            try:
                if events is not None:
                    events.close()
            finally:
                if child is not None:
                    try:
                        if child.stdin is not None and not child.stdin.closed:
                            child.stdin.close()
                        child.stdout.close()
                    finally:
                        if child.poll() is None:
                            child.terminate()
                        child.wait()
        finally:
            try:
                writer.close()
            finally:
                progress.close()
    print('{}: {} Bytes gelesen, {} Bytes unlesbar übersprungen.'.format(
        output, writer.manifest['bytes_read'], writer.manifest['bytes_unreadable']), file=sys.stderr)
    artifacts = writer.manifest['artifacts']
    print('{} Zusatzdateien gesichert, davon {} Debug-Dateien; {} Hinweise in manifest.json.'.format(
        len(artifacts), sum(item['role'] == 'debug' for item in artifacts),
        len(writer.manifest['warnings'])), file=sys.stderr)
    if args.refresh:
        print('SHA256-Abgleich: {} Bytes unverändert wiederverwendet, {} Bytes neu gelesen.'.format(
            writer.manifest['bytes_reused'], writer.manifest['bytes_refreshed']), file=sys.stderr)


def terminated(signum, frame):
    raise InterruptedError('Durch Signal {} abgebrochen'.format(signum))


if __name__ == '__main__':
    signal.signal(signal.SIGTERM, terminated)
    try:
        main()
    except (OSError, RuntimeError, ValueError, zlib.error, KeyboardInterrupt) as exc:
        print('Abbruch: {}. Ein bereits angelegter Ordner enthält höchstens einen Teildump; siehe manifest.json.'.format(exc), file=sys.stderr)
        sys.exit(1)
