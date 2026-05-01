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

/* Minimal repro for BTW2025 §4.1:
subquery in FULL OUTER JOIN ... ON must preserve unmatched rows on both sides
even when the ON predicate includes a correlated scalar subquery. */

(define repro_full_outer_subquery_on_setup (list
	"DROP TABLE IF EXISTS fooj_left"
	"DROP TABLE IF EXISTS fooj_right"
	"DROP TABLE IF EXISTS fooj_aux"
	"CREATE TABLE fooj_left (id INT PRIMARY KEY, note TEXT)"
	"CREATE TABLE fooj_right (id INT PRIMARY KEY, tag TEXT)"
	"CREATE TABLE fooj_aux (id INT PRIMARY KEY, keep_match INT)"
	"INSERT INTO fooj_left VALUES (1, 'L1'), (2, 'L2')"
	"INSERT INTO fooj_right VALUES (1, 'R1'), (2, 'R2'), (3, 'R3')"
	"INSERT INTO fooj_aux VALUES (1, 1), (2, 0), (3, 1)"
))

(define repro_full_outer_subquery_on_queries (list
	"SELECT l.id AS lid, r.id AS rid FROM fooj_left l FULL OUTER JOIN fooj_right r ON l.id = r.id AND COALESCE((SELECT a.keep_match FROM fooj_aux a WHERE a.id = COALESCE(l.id, r.id) LIMIT 1), 0) = 1 ORDER BY lid, rid"
))
