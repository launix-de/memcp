# SQL Support Gaps: fuchsbriefe

Analysis of SQL queries in `../fuchsbriefe/out` vs. memcp capabilities.

Some test cases are `noncritical: true`. Once a feature is implemented,
remove the flag from its tests to make them critical (blocking).

## High Priority

### UNION ALL set-level ORDER/LIMIT/OFFSET
- **Usage:** global `ORDER BY` / `LIMIT` after `UNION ALL` in read paths.
- **Status:** baseline UNION ALL is implemented (term parser + begin-concatenation + IN-subselect lowering + branch checks), but set-level ordering/limiting is intentionally blocked with clear errors.
- **Open design:** needs a dedicated shuffle/merge operator instead of ad-hoc materialization.
- **Concept doc:** `todos/union-sort.md`
- **Tests:** currently kept `noncritical` in `tests/70_union_all.yaml` until shuffle/merge exists.

## Medium Priority

### Type aliases: LONGTEXT, LONGBLOB, DATETIME
- **Usage:** 50 LONGTEXT columns, 1 LONGBLOB, 1 DATETIME in schema
- **Status:** TEXT/TIMESTAMP exist but these aliases are not mapped
- **Effort:** Small - parser type mapping (LONGTEXT->TEXT, LONGBLOB->TEXT, DATETIME->TIMESTAMP)
- **Tests:** `tests/43_dbcheck_ddl_compat.yaml` (type aliases section)

### DATE_FORMAT() compatibility
- **Usage:** Time formatting with MySQL format strings (%Y, %m, %d, etc.)
- **Status:** Function exists, needs verification that all MySQL format specifiers work correctly
- **Effort:** Test and fix

## Low Priority

### RAND() / RANDOM()
- **Usage:** Random ordering in queries
- **Effort:** Small - add to builtins
- **Tests:** `tests/45_math_date_functions.yaml` (RAND section)

### ENGINE=MEMORY behavior
- **Usage:** fail2ban table uses MEMORY engine (no persistence, data lost on restart)
- **Status:** MEMORY is parsed/mapped, verify non-persist behavior

### Prefix index length: INDEX(name(24))
- **Usage:** Some of 150+ indexes use prefix length
- **Status:** Parsed in CREATE INDEX, verify it doesn't error out

## Already Covered (no action needed)

- SELECT, INSERT, UPDATE, DELETE
- INSERT IGNORE
- ON DUPLICATE KEY UPDATE (syntax)
- VALUES() in ON DUPLICATE KEY UPDATE
- ON CONFLICT DO UPDATE (PostgreSQL)
- LEFT JOIN, INNER JOIN, CROSS JOIN
- WHERE with AND/OR/NOT, IN, BETWEEN, LIKE, IS NULL
- GROUP BY / HAVING
- ORDER BY ASC/DESC
- LIMIT / OFFSET
- CASE WHEN ... THEN ... END
- COALESCE(), CONCAT(), SUBSTR(), GROUP_CONCAT()
- COUNT(), SUM(), AVG(), MIN(), MAX()
- UNIX_TIMESTAMP(), CURRENT_DATE(), NOW()
- CAST()
- CREATE TABLE with PK, UNIQUE, FK, AUTO_INCREMENT
- Transactions (BEGIN/COMMIT/ROLLBACK)
- EXISTS / NOT EXISTS subqueries
- Scalar subqueries, IN (SELECT ...) subqueries
- UNION ALL baseline (without set-level ORDER/LIMIT/OFFSET)
- ENGINE=InnoDB/MyISAM mapping
- CHARACTER SET / COLLATE (parsed)
- CREATE/DROP DATABASE
- CREATE INDEX
- ALTER TABLE ADD/MODIFY/DROP COLUMN
- DECIMAL(precision, scale)
- Session variables (@var, SET @var = value)
