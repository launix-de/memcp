# Fixed bugs (2026-01-26)

The following bugs have been fixed and covered with test cases:

1. LIMIT/OFFSET bug with JOINs - offset/limit was passed to nested scans, causing wrong data
2. Column resolution bug - unqualified columns resolved to subquery tables instead of main table
3. REST Scheme backend - verified admin-only protection at /scm endpoint

Test cases: tests/19_subselect_order.yaml

---

# Missing SQL features for anim67or compatibility

Analysis of /home/carli/projekte/games/anim67or/out shows the following SQL features used
that are not yet fully supported by MemCP:

## EXISTS subqueries (high priority)
Used extensively in permission checks (Files.php):
```sql
EXISTS (SELECT ID FROM renderjob WHERE result = fop_files.ID)
EXISTS (SELECT ID FROM scene WHERE img = fop_files.ID AND ...)
```
Currently EXISTS always evaluates to empty or all rows.

## BETWEEN operator
Used in IDList.php for range queries:
```sql
WHERE col BETWEEN 1000 AND 3000
```

## NOT IN clause
Used for exclusion filters:
```sql
WHERE ID NOT IN (1, 3)
```

## IN with subquery
Used for permission checks:
```sql
WHERE ID IN (SELECT result FROM renderjob)
```

## Scalar subquery empty result
Scalar subqueries return NULL instead of 0 for COUNT(*) when no rows match.
Workaround: Use COALESCE((SELECT COUNT(*) ...), 0)

## Other features used (lower priority)
- LOCK TABLES (Cron.php) - not critical, used for concurrent cron protection
- INSERT IGNORE / ON CONFLICT DO NOTHING (Cron.php)
- Session variables SELECT @var := value (User.php)
- UPDATE with JOIN syntax (Files.php) - MySQL specific

Test cases: tests/20_anim67or_queries.yaml

---

# Old bug notes (reference)

SELECT
					(SELECT `ID` FROM fop_menu WHERE owner = `user`.`ID` AND link = 'http://localhost/supergollis/DataView/Karte/renderTabTable?page_karte=1&parent_request=%2Fsupergollis%2FTables%2FKarte%2Findex') AS `fop_favorites`,
					locale AS `locale`,
					secret_2fa AS `secret_2fa`,
					FALSE AS `isFirstUser`,
					TRUE AS `isLoggedIn`,
					FALSE AS `isGuest`,
					`ID` AS `ID`,
					`username` AS `username`,
					token AS `token`
				FROM `user` WHERE `ID` = '1'
trace 4.290581ms SELECT @fop_user := '1'
trace 52.332304ms SELECT t.* FROM (SELECT /* ID */ `ID` AS `ID`,

							/* author */ `author` AS `author`,

							/* author::canView */ `ref:author`.`author:canView` AS `author::canView`,

							/* author::description */ `ref:author`.`author:description` AS `author::description`,

							/* bild */ `bild` AS `bild`,

							/* date */ `date` AS `date`,

							/* farbe */ `farbe` AS `farbe`,

							/* name */ `name` AS `name`,

							/* operation::karte:delete */ TRUE AS `operation::karte:delete`,

							/* operation::karte:edit */ TRUE AS `operation::karte:edit`,

							/* operation::karte:view */ TRUE AS `operation::karte:view`,

							/* power */ ((((((`karte`.`power1`) + (`karte`.`power2`)) + (`karte`.`power3`)) + (`karte`.`power4`)) + (`karte`.`power5`)) + (`karte`.`power6`)) / (6) AS `power`,

							/* power1 */ `power1` AS `power1`,

							/* power2 */ `power2` AS `power2`,

							/* power2::vis */ `karte`.`typ` IN ('1') AS `power2::vis`,

							/* power3 */ `power3` AS `power3`,

							/* power3::vis */ `karte`.`typ` IN ('1') AS `power3::vis`,

							/* power4 */ `power4` AS `power4`,

							/* power4::vis */ `karte`.`typ` IN ('1') AS `power4::vis`,

							/* power5 */ `power5` AS `power5`,

							/* power5::vis */ `karte`.`typ` IN ('1') AS `power5::vis`,

							/* power6 */ `power6` AS `power6`,

							/* power6::vis */ `karte`.`typ` IN ('1') AS `power6::vis`,

							/* sammlung */ `sammlung` AS `sammlung`,

							/* sammlung::canView */ `ref:sammlung`.`sammlung:canView` AS `sammlung::canView`,

							/* sammlung::description */ `ref:sammlung`.`sammlung:description` AS `sammlung::description`,

							/* typ */ `typ` AS `typ`,

							1
							FROM `karte` `karte`
							LEFT JOIN (SELECT `ID` AS `sammlung:ID`, `sammlung`.`name` AS `sammlung:description`, (true) AS `sammlung:canView` FROM `sammlung`) `ref:sammlung` ON `ref:sammlung`.`sammlung:ID` = `sammlung`
							LEFT JOIN (SELECT `ID` AS `author:ID`, `golli`.`name` AS `author:description`, (true) AS `author:canView` FROM `golli`) `ref:author` ON `ref:author`.`author:ID` = `author`
							WHERE ((true))) AS t
							LEFT JOIN `golli` AS `sorter-author` ON t.`author` = `sorter-author`.`ID`
							WHERE
							 TRUE ORDER BY ((((((`t`.`power1`) + (`t`.`power2`)) + (`t`.`power3`)) + (`t`.`power4`)) + (`t`.`power5`)) + (`t`.`power6`)) / (6) DESC LIMIT 72 OFFSET 36

 - sobald ich limit/offset mit offset z.B. 35 benutze lädt er quatsch in die spalten (z.b. author -> NULL) [FIXED]
 - Name enthält den Namen des Autors, richtig wäre aber Name der Karte [FIXED]

test cases designen, bug fixen, make test -> verifizieren [DONE]

weitere bugs und TODOs (selbe testprozedur)
 - REST: Scheme backend darf nur für Admins erlaubt sein (bitte prüfen) [VERIFIED - already protected]
 - weitere SCHEMA queries aus den dbcheck.php Dateien

noch ein bug:

SELECT /* ID */ `ID` AS `ID`,

									/* ID */ `karte`.`ID` AS `ID`,

									/* author */ `karte`.`author` AS `author`,

									/* author::canView */ `ref:author`.`author:canView` AS `author::canView`,

									/* author::description */ `ref:author`.`author:description` AS `author::description`,

									/* bild */ `karte`.`bild` AS `bild`,

									/* date */ `karte`.`date` AS `date`,

									/* farbe */ `karte`.`farbe` AS `farbe`,

									/* name */ `karte`.`name` AS `name`,

									/* operation::create */ 1 AS `operation::create`,

									/* operation::delete */ true AS `operation::delete`,

									/* operation::edit */ true AS `operation::edit`,

									/* operation::index */ 1 AS `operation::index`,

									/* operation::view */ true AS `operation::view`,

									/* power */ ((((((`karte`.`power1`) + (`karte`.`power2`)) + (`karte`.`power3`)) + (`karte`.`power4`)) + (`karte`.`power5`)) + (`karte`.`power6`)) / (6) AS `power`,

									/* power1 */ `karte`.`power1` AS `power1`,

									/* power2 */ `karte`.`power2` AS `power2`,

									/* power2::canView */ `karte`.`typ` IN ('1') AS `power2::canView`,

									/* power3 */ `karte`.`power3` AS `power3`,

									/* power3::canView */ `karte`.`typ` IN ('1') AS `power3::canView`,

									/* power4 */ `karte`.`power4` AS `power4`,

									/* power4::canView */ `karte`.`typ` IN ('1') AS `power4::canView`,

									/* power5 */ `karte`.`power5` AS `power5`,

									/* power5::canView */ `karte`.`typ` IN ('1') AS `power5::canView`,

									/* power6 */ `karte`.`power6` AS `power6`,

									/* power6::canView */ `karte`.`typ` IN ('1') AS `power6::canView`,

									/* sammlung */ `karte`.`sammlung` AS `sammlung`,

									/* sammlung::canView */ `ref:sammlung`.`sammlung:canView` AS `sammlung::canView`,

									/* sammlung::description */ `ref:sammlung`.`sammlung:description` AS `sammlung::description`,

									/* typ */ `karte`.`typ` AS `typ`,

									1
								FROM `karte` `karte`
								LEFT JOIN (SELECT `ID` AS `sammlung:ID`, `sammlung`.`name` AS `sammlung:description`, (true) AS `sammlung:canView` FROM `sammlung`) `ref:sammlung` ON `ref:sammlung`.`sammlung:ID` = `karte`.`sammlung`
								LEFT JOIN (SELECT `ID` AS `author:ID`, `golli`.`name` AS `author:description`, (true) AS `author:canView` FROM `golli`) `ref:author` ON `ref:author`.`author:ID` = `karte`.`author`
								WHERE (`ID` = '2')

die query lädt den falschen Datensatz. wird das WHERE ID=2 etwa verschluckt? [FIXED - column resolution issue]
