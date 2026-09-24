#!/usr/bin/env python3
# Copyright (C) 2026 Carl-Philip Hänsch
# SPDX-License-Identifier: GPL-3.0-or-later
"""Compare public SQL behaviour across an old-writer/new-reader upgrade.

Every case is executed repeatedly. This both compares cold and warm results and
gives adaptive caches enough observed work to be admitted naturally. The test
never inspects or names planner-owned tables, columns or operators.
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


def cases(changed=False):
    unchanged = not changed
    return [
        ('group aggregate', 12,
         'SELECT bucket, COUNT(*), SUM(value) FROM up_helper GROUP BY bucket ORDER BY bucket',
         '1\t2\t30\n2\t2\t70' if unchanged else '1\t2\t35\n2\t2\t70'),
        ('large group aggregate', 12,
         'SELECT tenant_id, COUNT(*), SUM(amount) FROM up_group_events GROUP BY tenant_id ORDER BY tenant_id',
         '1\t3000\t26992\n2\t3000\t27000' if unchanged else
         '1\t3000\t27092\n2\t3000\t27000'),
        ('computed group order', 12,
         'SELECT bucket, SUM(value) FROM up_helper GROUP BY bucket ORDER BY SUM(value) + 1 DESC',
         '2\t70\n1\t30' if unchanged else '2\t70\n1\t35'),
        ('expression group keys', 12,
         'SELECT bucket+10, SUM(value) FROM up_helper GROUP BY bucket+10 ORDER BY bucket+10',
         '11\t30\n12\t70' if unchanged else '11\t35\n12\t70'),
        ('correlated range and order', 12,
         'SELECT w.id, (SELECT e.id FROM up_group_events e '
         'WHERE e.tenant_id=w.tenant_id AND e.happened_at<=w.range_to '
         'ORDER BY e.happened_at DESC LIMIT 1) '
         'FROM up_group_windows w ORDER BY w.id',
         '1\t1000\n2\t1500\n3\t3001\n4\t3001' if unchanged else
         '1\t998\n2\t1500\n3\t3001\n4\t3001'),
        ('ordered scalar aggregate', 6,
         'SELECT p.id, (SELECT h.value FROM up_helper h WHERE h.bucket=p.id '
         'ORDER BY h.id DESC LIMIT 1) FROM up_helper_parent p ORDER BY p.id',
         '1\t20\n2\t40'),
        ('opposing ordered windows', 6,
         'SELECT id, '
         'ROW_NUMBER() OVER (PARTITION BY bucket ORDER BY id ASC), '
         'ROW_NUMBER() OVER (PARTITION BY bucket ORDER BY id DESC), '
         'LAG(value) OVER (PARTITION BY bucket ORDER BY id ASC), '
         'LAG(value) OVER (PARTITION BY bucket ORDER BY id DESC) '
         'FROM up_helper ORDER BY id',
         ('1\t1\t2\tNULL\t20\n2\t2\t1\t10\tNULL\n'
          '3\t1\t2\tNULL\t40\n4\t2\t1\t30\tNULL') if unchanged else
         ('1\t1\t2\tNULL\t20\n2\t2\t1\t15\tNULL\n'
          '3\t1\t2\tNULL\t40\n4\t2\t1\t30\tNULL')),
        ('partition aggregate', 6,
         'SELECT id, SUM(value) OVER (PARTITION BY bucket) FROM up_helper ORDER BY id',
         '1\t30\n2\t30\n3\t70\n4\t70' if unchanged else
         '1\t35\n2\t35\n3\t70\n4\t70'),
        ('correlated lookup order', 6,
         'SELECT d.id FROM up_helper_driver d ORDER BY '
         '(SELECT f.stamp FROM up_helper_file f WHERE f.id=d.file_id LIMIT 1) DESC, '
         'd.id DESC LIMIT 3',
         '4000\n3999\n3998' if unchanged else '1\n4000\n3999'),
    ]


def run_cases(changed=False, oracle=None):
    results = {}
    for name, repetitions, sql, expected in cases(changed):
        print(f'checking query: {name} ({repetitions} executions)', flush=True)
        outputs = []
        for execution in range(1, repetitions + 1):
            actual = query(sql)
            if actual != expected:
                raise AssertionError(
                    f'{name} execution {execution}: expected {expected!r}, got {actual!r}')
            outputs.append(actual)
        results[name] = outputs
    if oracle is not None and results != oracle:
        raise AssertionError('public SQL results differ from the old-writer oracle')
    return results


def load_oracle(path):
    value = json.loads(pathlib.Path(path).read_text(encoding='utf-8'))
    if value.get('format') != 'memcp-upgrade-query-results-v1':
        raise ValueError('unknown query oracle format')
    return value['results']


def write_oracle(path, results):
    with pathlib.Path(path).open('x', encoding='utf-8') as output:
        json.dump({'format': 'memcp-upgrade-query-results-v1', 'results': results},
                  output, ensure_ascii=False, sort_keys=True, indent=2)
        output.write('\n')


if mode == 'record':
    write_oracle(args[0], run_cases())
elif mode == 'check':
    run_cases(oracle=load_oracle(args[0]))
elif mode in ('mutate', 'cold-mutate'):
    oracle = load_oracle(args[0])
    # Cold mutation happens before any query on this process; warm mutation
    # happens after the repeated baseline has exercised adaptive caches.
    if mode == 'mutate':
        run_cases(oracle=oracle)
    query('UPDATE up_helper SET value=15 WHERE id=1')
    query('UPDATE up_helper_file SET stamp=5000 WHERE id=1')
    query('UPDATE up_group_events SET amount=101 WHERE id=1')
    query('UPDATE up_group_events SET happened_at=1001 WHERE id=1000')
    run_cases(changed=True)
    query('UPDATE up_helper SET value=10 WHERE id=1')
    query('UPDATE up_helper_file SET stamp=1 WHERE id=1')
    query('UPDATE up_group_events SET amount=1 WHERE id=1')
    query('UPDATE up_group_events SET happened_at=999 WHERE id=1000')
    run_cases(oracle=oracle)
else:
    raise ValueError('unknown mode: ' + mode)
