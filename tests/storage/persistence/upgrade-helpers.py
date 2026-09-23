#!/usr/bin/env python3
# Copyright (C) 2026 Carl-Philip Hänsch
# SPDX-License-Identifier: GPL-3.0-or-later
"""Exercise real planner helpers across an old-writer/new-reader boundary.

The writer must execute these queries and persist every required helper family.
The reader may migrate, discard or rename helpers, but must preserve SQL results.
Never synthesize old schema JSON: inspect only what the old server wrote.
"""
import json
import pathlib
import subprocess
import sys

mode, port, *args = sys.argv[1:]
client = ['mysql', '-h', '127.0.0.1', '-P', port, '-u', 'root', '-padmin',
          '-N', '-B', '--raw', 'memcp-tests']


def query(sql):
    try:
        return subprocess.run(client + ['-e', sql], capture_output=True,
                              check=True, timeout=30).stdout.decode().strip()
    except subprocess.CalledProcessError as error:
        raise RuntimeError(f'{sql}\n{error.stderr.decode()}') from error


def checks(changed=False):
    # Distinct keys, values and ordering make wrong-but-valid layouts observable.
    cases = [
        ('group aggregate',
         'SELECT bucket, COUNT(*), SUM(value) FROM up_helper GROUP BY bucket ORDER BY bucket',
         '1\t2\t30\n2\t2\t70' if not changed else '1\t2\t35\n2\t2\t70'),
        ('computed group order',
         'SELECT bucket, SUM(value) FROM up_helper GROUP BY bucket ORDER BY SUM(value) + 1 DESC',
         '2\t70\n1\t30' if not changed else '2\t70\n1\t35'),
        ('expression group keys',
         'SELECT bucket+10, SUM(value) FROM up_helper GROUP BY bucket+10 ORDER BY bucket+10',
         '11\t30\n12\t70' if not changed else '11\t35\n12\t70'),
        ('ordered scalar aggregate',
         'SELECT p.id, (SELECT h.value FROM up_helper h WHERE h.bucket=p.id ORDER BY h.id DESC LIMIT 1) FROM up_helper_parent p ORDER BY p.id',
         '1\t20\n2\t40'),
        ('partitioned row number',
         'SELECT id, ROW_NUMBER() OVER (PARTITION BY bucket ORDER BY id) FROM up_helper ORDER BY id',
         '1\t1\n2\t2\n3\t1\n4\t2'),
        ('partition aggregate',
         'SELECT id, SUM(value) OVER (PARTITION BY bucket) FROM up_helper ORDER BY id',
         '1\t30\n2\t30\n3\t70\n4\t70' if not changed else '1\t35\n2\t35\n3\t70\n4\t70'),
        ('window offset',
         'SELECT id, LAG(value) OVER (PARTITION BY bucket ORDER BY id) FROM up_helper ORDER BY id',
         '1\tNULL\n2\t10\n3\tNULL\n4\t30' if not changed else '1\tNULL\n2\t15\n3\tNULL\n4\t30'),
        ('canonical lookup column',
         'SELECT id FROM up_helper_driver ORDER BY `.lookup:upgrade_file_stamp` DESC, id DESC LIMIT 3',
         '4000\n3999\n3998' if not changed else '1\n4000\n3999'),
    ]
    for name, sql, expected in cases:
        print(f'checking helper: {name}', flush=True)
        actual = query(sql)
        if actual != expected:
            raise AssertionError(f'{name}: expected {expected!r}, got {actual!r}')
        print(f'helper {name}: OK', flush=True)


def inventory(data, output):
    schema = json.loads((pathlib.Path(data) / 'memcp-tests/schema.json').read_text())
    families = {name: [] for name in ('group-keytable', 'range-boundaries',
        'range-state', 'aggregate-column', 'computed-order', 'lookup-column', 'window-orc')}
    for name, table in schema['tables'].items():
        columns = table['Columns']
        if name.startswith('.grp:up_'):
            if any(c['Name'].startswith('range_') for c in columns):
                families['range-boundaries'].append(name)
            else:
                families['group-keytable'].append(name)
        for col in columns:
            if not (name.startswith('up_') or name.startswith('.grp:up_')):
                continue
            column = col['Name']
            kind = next((kind for prefix, kind in (
                ('agg_range_state_', 'range-state'), ('agg_', 'aggregate-column'),
                ('ord_', 'computed-order'), ('.lookup:', 'lookup-column'),
                ('__orc_', 'window-orc')) if column.startswith(prefix)), None)
            if kind:
                if not col.get('IsTemp'):
                    raise AssertionError(f'{name}.{column}: expected temporary column')
                families[kind].append(f'{name}.{column}')
    missing = [name for name, objects in families.items() if not objects]
    if missing:
        raise AssertionError(f'old writer did not persist required helper families: {missing}')
    with open(output, 'x') as out:
        json.dump({'families': families, 'schema': schema}, out, indent=2)
    print('persisted old-writer helper coverage: ' + ', '.join(
        f'{name}={len(objects)}' for name, objects in families.items()))


if mode == 'check':
    checks()
elif mode in ('mutate', 'cold-mutate'):
    # Re-execute before/after dependency changes; restore application rows so
    # the existing complete old-writer snapshot remains the authoritative oracle.
    if mode == 'mutate':
        checks()
    query('UPDATE up_helper SET value=15 WHERE id=1')
    query('UPDATE up_helper_file SET stamp=5000 WHERE id=1')
    checks(True)
    query('UPDATE up_helper SET value=10 WHERE id=1')
    query('UPDATE up_helper_file SET stamp=1 WHERE id=1')
    checks()
elif mode == 'inventory':
    inventory(*args)
else:
    raise ValueError('unknown mode: ' + mode)
