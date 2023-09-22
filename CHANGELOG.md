0.1.2
=====

- added Dockerfile
- added function help
- storage function: scan_order

0.1.1
=====

- IO functions: password
- user table for mysql auth
- mysql and REST API check for username/password

0.1.0
=====

- basic scheme functions: quote, eval, if, and, or, match, define/set, lambda, begin, error, symbol, list
- arithmetic scheme functions: +, -, *, /, <=, <, >=, >, equal?, !/not
- scheme string functions: simplify, strlen, concat, toLower, toUpper, split
- scheme list functions: append, cons, car, cdr, merge, has?, filter, map, reduce
- scheme dictionary functions: filter_assoc, map_assoc, reduce_assoc, set_assoc, has_assoc?, merge_assoc
- IO functions: print, import, load, serve, mysql
- storage functions: scan, createdatabase, dropdatabase, createtable, droptable, insert, stat, rebuild, loadCSV, loadJSON
- storage types: SCMER, int, sequence, string, dictionary, float
- SQL: support for SELECT * FROM, CREATE DATABASE, CREATE TABLE, SHOW DATABASES, SHOW TABLES, INSERT INTO
