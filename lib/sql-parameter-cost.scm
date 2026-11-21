/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* These are compiler candidates, not cache entries. Compilation is charged
only when work must be done; reusing an existing callable has no compile cost.
Unknown execution work cannot establish a specialization benefit. */
(define sql_parameter_candidate_cost (lambda (variant reused session tx)
	(begin
		(define info (nth variant 3))
		(define modeled (qassoc_get (qassoc_get info (quote execution_cost) nil)
			(quote execution_ns) nil))
		(define observation (qassoc_get info (quote observation) nil))
		(define observed (if (nil? observation) nil (observation "execution_ns")))
		(define work_cost (parameter_plan_work_cost (qassoc_get info (quote work_profile) nil) session tx))
		/* Full Cartesian work admits reuse; it must not erase the chosen
		physical plans' relative costs. Add only previously unpriced bytes. */
		(define execution (if (number? modeled)
			(max (+ modeled (if (nil? work_cost) 0 (cadr work_cost))) (coalesceNil observed 0)) nil))
		(if (not (number? execution)) nil
			(planner_cost 0 execution 0 0 0 0 0
				(if reused 0 (qassoc_get info (quote compile_ns) 0)) nil 0.5)))))

(define sql_parameter_specialization_preferred? (lambda (specialized generic generic_reused session tx)
	(begin
		(define specialized_cost (sql_parameter_candidate_cost specialized false session tx))
		(define generic_cost (sql_parameter_candidate_cost generic generic_reused session tx))
		(and (not (nil? specialized_cost)) (not (nil? generic_cost))
			/* A shorter compilation alone does not justify retaining a plan for
			one value. Its execution must repay the additional compilation. */
			(< (qassoc_get specialized_cost (quote execution_ns) 0)
				(qassoc_get generic_cost (quote execution_ns) 0))
			(planner_cost_better? specialized_cost generic_cost)))))

/* Only parameterized entries pay query-boundary timing. The dispatch closure
retains scalar observations, never the executing request, transaction or rows.
An observed model miss revokes reuse on the next boundary; it cannot authorize
specialization without a fresh candidate cost comparison. */
(define sql_parameter_generic? (lambda (variant)
	(begin
		(define mode (qassoc_get (nth variant 3) (quote parameter_mode) nil))
		(or (equal? mode "generic") (equal? mode "parameterized")))))

(define sql_parameter_reusable? (lambda (info session tx)
	(begin
		(define budget (qassoc_get info (quote recompile_ns) nil))
		(define work_cost (parameter_plan_work_cost (qassoc_get info (quote work_profile) nil) session tx))
		(define work (if (nil? work_cost) nil (car work_cost)))
		(define observation (qassoc_get info (quote observation) nil))
		(and (number? budget) (number? work) (<= work budget)
			(<= (coalesceNil (observation "execution_ns") 0) budget)))))

(define sql_parameter_execute (lambda (info plan session tx resultrow resultfields)
	(begin
		(define started (nanotime))
		(define result (plan session tx resultrow resultfields))
		((qassoc_get info (quote observation) nil) "execution_ns" (- (nanotime) started))
		result)))

(define sql_parameter_choose_candidate (lambda (specialized entry variants parse_fn schema parse_query policy session tx)
	(begin
		(define specialized_info (nth specialized 3))
		/* A miss only permits evaluation. Neither miss count nor the existing
		variant cap participates in the cost decision. Unknown work declines. */
		(define prior_compiles (map (filter variants (lambda (variant)
			(not (sql_parameter_generic? variant))))
			(lambda (variant) (qassoc_get (nth variant 3) (quote compile_ns) 0))))
		(define budget (reduce prior_compiles min
			(qassoc_get specialized_info (quote compile_ns) 0)))
		(define work_cost (parameter_plan_work_cost (qassoc_get specialized_info (quote work_profile) nil) session tx))
		(define work (if (nil? work_cost) nil (car work_cost)))
		(define previous_observation (entry "parameter_observation"))
		(define observed (if (nil? previous_observation) 0
			(coalesceNil (previous_observation "execution_ns") 0)))
		/* Decline an unpriced or already expensive domain before compiling a
		second candidate. Large flat scans retain the original miss cost too. */
		(if (or (empty_list? (qassoc_get specialized_info (quote parameters) '()))
			(not (number? work)) (> work budget) (> observed budget)
			(nil? (sql_parameter_candidate_cost specialized false session tx))) specialized
			(begin
				(define generic (sql_compile_queryplan_variant parse_fn schema parse_query policy session tx "parameterized"))
				/* Carry actual generic execution evidence across recompilation. A
				fresh scalar cell must not erase the observation which caused this
				miss and select the same demonstrably expensive work again. */
				(define observation (if (nil? previous_observation) (newsession) previous_observation))
				(entry "parameter_observation" observation)
				(define info (qassoc_set (qassoc_set (nth generic 3)
					(quote recompile_ns) budget) (quote observation) observation))
				(define candidate (list (car generic) (cadr generic) (nth generic 2) info))
				(if (and (not (nil? (sql_parameter_candidate_cost candidate false session tx)))
					(sql_parameter_reusable? info session tx)
					(not (sql_parameter_specialization_preferred? specialized candidate false session tx)))
					(list (sql_queryplan_compile_formula tx
						(list (quote and)
							(if (expression_equal? (car generic) true) true
								(list (quote apply) (car generic)
									(list (quote list) (quote session) (quote tx) (quote resultrow) (quote resultfields))))
							(list (quote sql_parameter_reusable?) (list (quote quote) info) (quote session) (quote tx))))
						(cadr generic) (nth generic 2) info)
					specialized))))))
