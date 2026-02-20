# SQL Support Gaps: fuchsbriefe

Analysis of SQL queries in `../fuchsbriefe/out` vs. memcp capabilities.

All test cases are currently `noncritical: true`. Once a feature is implemented,
remove the `noncritical: true` flag from its tests to make them critical (blocking).

## High Priority

### UNION ALL
- **Usage:** 18+ instances in `Files.php` combining file sources from multiple tables
- **Implementation:** Just a `(begin)` block with two scans
- **UNION ALL inside IN subselect:** compile to `(or (scan...) (scan...))`
- **Effort:** Medium - parser + queryplan adjustment
- **Tests:** `tests/70_union_all.yaml` (standalone), `tests/41_in_subquery.yaml` (IN with UNION ALL)

### VALUES() in ON DUPLICATE KEY UPDATE
- **Usage:** `ON DUPLICATE KEY UPDATE col = VALUES(col)` in Outlook.php, Imap.php, Push.php, Login.php, Index.php, and many Table classes
- **Status:** ON DUPLICATE KEY UPDATE is supported, but `VALUES(col)` referencing the to-be-inserted values may not work
- **Effort:** Small - add function to builtins
- **Tests:** already covered in `tests/29_mysql_upsert.yaml` (verify VALUES() works)

## Medium Priority

### IFNULL() function
- **Usage:** NULL handling across multiple queries
- **Status:** COALESCE() exists but IFNULL(val, default) as MySQL alias is missing
- **Effort:** Small - alias to COALESCE in builtins
- **Tests:** `tests/60_case_coalesce_nullif.yaml` (IFNULL section)

### IF(cond, true_val, false_val) function
- **Usage:** Conditional logic in queries (MySQL-style 3-arg IF)
- **Effort:** Small - 3-arg conditional in builtins
- **Tests:** `tests/60_case_coalesce_nullif.yaml` (IF section)

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
- ENGINE=InnoDB/MyISAM mapping
- CHARACTER SET / COLLATE (parsed)
- CREATE/DROP DATABASE
- CREATE INDEX
- ALTER TABLE ADD/MODIFY/DROP COLUMN
- DECIMAL(precision, scale)
- Session variables (@var, SET @var = value)
