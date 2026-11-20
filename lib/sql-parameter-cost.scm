/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* These are compiler candidates, not cache entries. Compilation is charged
only when work must be done; reusing an existing callable has no compile cost.
Unknown execution work cannot establish a specialization benefit. */
(define sql_parameter_candidate_cost (lambda (variant reused)
	(begin
		(define info (nth variant 3))
		(define execution (qassoc_get (qassoc_get info (quote execution_cost) nil)
			(quote execution_ns) nil))
		(if (not (number? execution)) nil
			(planner_cost 0 execution 0 0 0 0 0
				(if reused 0 (qassoc_get info (quote compile_ns) 0)) nil 0.5)))))

(define sql_parameter_specialization_preferred? (lambda (specialized generic generic_reused)
	(begin
		(define specialized_cost (sql_parameter_candidate_cost specialized false))
		(define generic_cost (sql_parameter_candidate_cost generic generic_reused))
		(and (not (nil? specialized_cost)) (not (nil? generic_cost))
			/* A shorter compilation alone does not justify retaining a plan for
			one value. Its execution must repay the additional compilation. */
			(< (qassoc_get specialized_cost (quote execution_ns) 0)
				(qassoc_get generic_cost (quote execution_ns) 0))
			(planner_cost_better? specialized_cost generic_cost)))))
