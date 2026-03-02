#!/usr/bin/env python3
"""Wrapper to run test runner with unbuffered output."""
import sys, os, builtins
os.environ['PYTHONUNBUFFERED'] = '1'
original_print = builtins.print
def flushed_print(*args, **kwargs):
    kwargs['flush'] = True
    original_print(*args, **kwargs)
builtins.print = flushed_print
sys.argv = ['run_sql_tests.py'] + sys.argv[1:]
from run_sql_tests import main
main()
