/*
Copyright (C) 2026 Carl-Philip Hänsch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/* Minimal repro for correlated scalar subqueries over an outer grouped domain.
The outer grouped derived table alias `o` must remain a logical outer table
binding for dependent domain construction; it must not collapse to a materialized
table-name string while building the helper domain query. */

(define repro66_correlated_group_domain_setup (list
	"DROP TABLE IF EXISTS cgd_state"
	"CREATE TABLE cgd_state (ID INT PRIMARY KEY, parent INT, grp INT)"
	"INSERT INTO cgd_state VALUES (1, NULL, 10), (2, 1, 10), (3, NULL, 20), (4, 1, 20), (5, 3, 20)"
))

(define repro66_correlated_group_domain_queries (list
	"SELECT o.grp, COALESCE((SELECT SUM(i.cnt) FROM (SELECT s2.parent, s2.grp, COUNT(*) AS cnt FROM cgd_state s2 GROUP BY s2.parent, s2.grp) i WHERE i.parent = o.sample_id AND i.grp = o.grp), 0) AS score FROM (SELECT grp, MIN(ID) AS sample_id FROM cgd_state GROUP BY grp) o ORDER BY o.grp"
	"EXPLAIN IR SELECT o.grp, COALESCE((SELECT SUM(i.cnt) FROM (SELECT s2.parent, s2.grp, COUNT(*) AS cnt FROM cgd_state s2 GROUP BY s2.parent, s2.grp) i WHERE i.parent = o.sample_id AND i.grp = o.grp), 0) AS score FROM (SELECT grp, MIN(ID) AS sample_id FROM cgd_state GROUP BY grp) o ORDER BY o.grp"
))
