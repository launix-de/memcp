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

/* Minimal repro for nested correlated subselect column resolution.
The inner scalar subselect must resolve `ev.vehicle` from the middle scope,
not duplicate across unrelated rows. */

(define repro66_nested_subselect_column_as_table_setup (list
	"DROP TABLE IF EXISTS nsc_doc"
	"DROP TABLE IF EXISTS nsc_event"
	"DROP TABLE IF EXISTS nsc_vehicle"
	"CREATE TABLE nsc_vehicle (ID INT PRIMARY KEY, name TEXT, public INT)"
	"CREATE TABLE nsc_event (ID INT PRIMARY KEY, vehicle INT, description TEXT)"
	"CREATE TABLE nsc_doc (ID INT PRIMARY KEY, subject TEXT, ref_event INT)"
	"INSERT INTO nsc_vehicle VALUES (1, 'Car A', 1), (2, 'Car B', 0)"
	"INSERT INTO nsc_event VALUES (1, 1, 'Oil change'), (2, 2, 'Tire swap')"
	"INSERT INTO nsc_doc VALUES (1, 'Invoice', 1), (2, 'Report', 2), (3, 'Note', NULL)"
))

(define repro66_nested_subselect_column_as_table_queries (list
	"SELECT d.ID, d.subject, COALESCE((SELECT (COALESCE((SELECT v.name FROM nsc_vehicle v WHERE v.ID = ev.vehicle LIMIT 1), 'unknown')) FROM nsc_event ev WHERE ev.ID = d.ref_event LIMIT 1), 'none') AS vehicle_name FROM nsc_doc d ORDER BY d.ID"
))
