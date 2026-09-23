#!/usr/bin/env python3
# Copyright (C) 2026 Carl-Philip Hänsch
# SPDX-License-Identifier: GPL-3.0-or-later
"""Validate the upgrade gate against unversioned range and group layouts.

Usage: upgrade-helper-negative.py BUILT_CANDIDATE OUTPUT_DIRECTORY
Runs the same lifecycle script as CI. The altered algorithm must work on a
fresh database, fail on an old writer's schema, then pass with a new cache ID.
The mutation is restricted to disposable copies of the candidate's Scheme lib.
"""
import os
import pathlib
import shutil
import signal
import subprocess
import sys

candidate = pathlib.Path(sys.argv[1]).resolve()
root = pathlib.Path(sys.argv[2]).resolve()
root.mkdir()  # Never overwrite an earlier result or source tree.


def variant(name, family, versioned):
    tree = root / name
    tree.mkdir()
    for entry in candidate.iterdir():
        if entry.name == 'lib':
            shutil.copytree(entry, tree / 'lib')
        elif entry.name not in ('.git', '.worktrees'):
            (tree / entry.name).symlink_to(entry)
    path = tree / 'lib/queryplan-physical-expr.scm'
    code = path.read_text()
    if family == 'range':
        old = '(if (equal? axis 0) "range" (concat "range_" axis))'
        new = '(if (equal? axis 0) "range_next" (concat "range_next_" axis))'
        version, next_version = '":range-v1"', '":range-negative-control-v2"'
    else:
        old, new = '(concat "k" i)', '(concat "key_next_" i)'
        # Scalar/constant group producers also have literal references to k0.
        for library in (tree / 'lib').glob('*.scm'):
            library.write_text(library.read_text().replace('"k0"', '"key_next_0"'))
        code = path.read_text()
        version, next_version = 'canonical-group-keytable-v7', 'canonical-group-keytable-negative-v8'
    if code.count(old) != 1 or code.count(version) != 1:
        raise AssertionError(f'{family} mutation anchor changed; update the negative control')
    code = code.replace(old, new)
    if versioned:
        code = code.replace(version, next_version)
        if family == 'group':
            code = code.replace('canonical-range-group-keytable-v3', 'canonical-range-group-keytable-negative-v4')
    path.write_text(code)
    return tree


for family in ('range', 'group'):
    broken = variant(f'{family}-unversioned', family, False)
    fixed = variant(f'{family}-versioned', family, True)
    for scenario, base, reader, must_pass in (
            ('fresh-layout', broken, broken, True),
            ('old-layout-without-migration', candidate, broken, False),
            ('old-layout-with-versioning', candidate, fixed, True)):
        name = f'{family}-{scenario}'
        workspace = root / name
        workspace.mkdir()
        (workspace / 'base').symlink_to(base)
        (workspace / 'candidate').symlink_to(reader)
        env = dict(os.environ, GITHUB_WORKSPACE=str(workspace))
        with (workspace / 'run.log').open('w') as log:
            process = subprocess.Popen(
                ['bash', str(candidate / 'tests/storage/persistence/upgrade-run.sh')],
                cwd=workspace, env=env, stdout=log, stderr=subprocess.STDOUT,
                start_new_session=True)
            try:
                status = process.wait(timeout=300)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
                raise
        output = (workspace / 'run.log').read_text()
        if must_pass and status != 0:
            raise AssertionError(f'{name} failed: {workspace / "run.log"}\n{output[-4000:]}')
        if not must_pass:
            # A failed setup or unrelated error is not evidence of a working gate.
            expected = ('MISMATCH: Range group cache survives upgrade' if family == 'range'
                        else 'checking helper: group aggregate')
            candidate_output = output.split('upgrade phase: candidate-upgrade', 1)
            if (status == 0 or len(candidate_output) != 2
                    or expected not in candidate_output[1]
                    or 'Column does not exist:' not in candidate_output[1]
                    or ('range_next_from_kind' if family == 'range' else 'key_next_0') not in candidate_output[1]):
                raise AssertionError(f'missing expected {family} upgrade failure: {output[-4000:]}')
        print(f'{name}: expected {"PASS" if must_pass else "FAIL"}, exit={status}', flush=True)
