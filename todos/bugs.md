# Unfinished SQL Features

## Correlated EXISTS in WHERE clause
Nested scan closure issue - inner scan lambda cannot access outer row values.
```sql
SELECT * FROM customers c
WHERE EXISTS (SELECT 1 FROM orders o WHERE o.customer_id = c.ID)
```
Test cases: tests/40_exists_subquery.yaml (noncritical)

## IN/NOT IN with subquery (table scan)
Same nested scan closure issue as EXISTS.
```sql
SELECT * FROM fop_files
WHERE ID IN (SELECT result FROM renderjob)
```
```sql
SELECT * FROM fop_files
WHERE ID NOT IN (SELECT result FROM renderjob)
```
Test cases: tests/41_in_subquery.yaml (noncritical)

## Session variables
MySQL-style session variable assignment.
```sql
SELECT @fop_user := '1'
```
```sql
SET @var = 5; SELECT @var
```

## INSERT IGNORE / ON CONFLICT DO NOTHING
Skip insert on duplicate key without error.
```sql
INSERT IGNORE INTO table (id, name) VALUES (1, 'test')
```
```sql
INSERT INTO table (id, name) VALUES (1, 'test') ON CONFLICT DO NOTHING
```

## UPDATE with JOIN
MySQL-specific multi-table update syntax.
```sql
UPDATE files f
JOIN users u ON f.owner = u.ID
SET f.status = 'active'
WHERE u.role = 'admin'
```

## LOCK TABLES (low priority)
Used for concurrent cron protection.
```sql
LOCK TABLES cron WRITE
```
```sql
UNLOCK TABLES
```
