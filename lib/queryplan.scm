/*
Copyright (C) 2023, 2024, 2026  Carl-Philip Haensch

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

/* Keep planner modules in pipeline order. Definitions are intentionally
loaded into the shared Scheme namespace for compatibility with existing
queryplan.scm importers. */
(import "queryplan-logical.scm")
(import "queryplan-optimize.scm")
(import "queryplan-physical-scan.scm")
(import "queryplan-physical-expr.scm")
/* Forward stub: prepare_physical_queryplan calls sql_type_annotate_ir, whose
real definition (sql-types.scm) needs helpers from queryplan-physical-plan.scm
and is loaded right after it. */
(define sql_type_annotate_ir (lambda (ir) ir))
(import "queryplan-physical-plan.scm")
(import "sql-types.scm")
