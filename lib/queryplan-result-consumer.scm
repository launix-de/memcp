/* Copyright (C) 2026 Carl-Philip Hänsch
SPDX-License-Identifier: GPL-3.0-or-later */

/* An output reduction is an explicit physical consumer contract. It carries
actual immutable callable values and an actual initial value/factory, never
caller expressions transplanted below another lambda's parameter bindings.
Factories provide fresh mutable state at each native reducer boundary.
Only the final output consumer receives it; prepares and probes retain their
own contracts. nil retains the existing effect-only output API. */
(define make_result_reduction (lambda (reducer initial combine)
	(if (or (nil? reducer) (nil? combine)) (error "result reduction requires a reducer and shard combiner")
		(list 'reduce reducer initial combine))))
(define result_consumer_reducer (lambda (consumer) (if (nil? consumer) nil (list 'quote (nth consumer 1)))))
(define result_consumer_combine (lambda (consumer) (if (nil? consumer) nil (list 'quote (nth consumer 3)))))
(define result_consumer_fold (lambda (consumer accumulator value)
	(if (nil? consumer) value (list (result_consumer_reducer consumer) accumulator value))))
(define result_consumer_scalar_initial? (lambda (initial)
	(or (nil? initial) (number? initial) (string? initial)
		(identical? initial true) (identical? initial false))))
(define result_consumer_initialize (lambda (consumer)
	(if (nil? consumer) nil (begin
		(define initial (nth consumer 2))
		/* Keep factory invocation at execution, even when its actual callable
		is compile-time constant. The native owner is shared with reducers. */
		(if (result_consumer_scalar_initial? initial) initial
			(list 'reducer_initial_value (list 'quote initial)))))))
(define result_consumer_initial (lambda (consumer)
	(if (nil? consumer) nil (begin
		(define initial (nth consumer 2))
		/* Self-evaluating scalar values need no initializer closure. Other
		values keep their actual runtime kind through the existing init owner. */
		(if (result_consumer_scalar_initial? initial) initial
			(list 'lambda '() (result_consumer_initialize consumer)))))))
(define result_consumer_single (lambda (consumer value)
	(result_consumer_fold consumer (result_consumer_initialize consumer) value)))
/* The join pipeline reserves a symbol for an absent contribution. Its native
reducers therefore receive one bounded state/value slot, so an arbitrary user
accumulator or callback result cannot be mistaken for that control marker. */
(define result_consumer_join_reducer (lambda (consumer combine)
	(list 'lambda (list 'state 'value)
		(list 'list (list (if combine (result_consumer_combine consumer) (result_consumer_reducer consumer))
			(list 'car 'state) (list 'car 'value))))))
(define result_consumer_join_neutral (lambda (consumer)
	/* Join recipes compose this value expression into their bounded match
	state too. Native scan constructors wrap it at their initializer boundary. */
	(list 'list (result_consumer_initialize consumer))))
/* Only cleanup code lies within the new result binding. Inspect that bounded
code without expanding opaque values; exact AST equality also sees SourceInfo
symbols. Quoted data is inert, and bound names are conservatively excluded too. */
(define result_consumer_mentions_symbol? (lambda (code name) (match code
	((symbol quote) _data) false
	(cons head tail) (or (result_consumer_mentions_symbol? head name)
		(reduce tail (lambda (found part) (or found (result_consumer_mentions_symbol? part name))) false))
	_ (expression_equal? code name))))
(define result_consumer_temporary (lambda (code index) (begin
	(define name (symbol (concat "__completed_consumer_result_" index)))
	(if (result_consumer_mentions_symbol? code name)
		(result_consumer_temporary code (+ index 1)) name))))
(define result_consumer_with_cleanup (lambda (prelude value cleanup) (begin
	(define tail (if (empty_list? cleanup) value (begin
		(define name (result_consumer_temporary cleanup 0))
		(list (list 'lambda (list name) (cons '!begin (append cleanup name))) value))))
	(if (empty_list? prelude) tail (cons '!begin (append prelude tail))))))
/* Final scan consumers retain the native scan's accumulator, initializer and
shard-combine boundary. This constructor never rewrites an emitted program or
looks inside preparation/probe callbacks. Each caller opts in at its real sink. */
(define result_consumer_scan_mapper (lambda (consumer mapper) (match mapper
	((symbol lambda) (cons accumulator parameters) body)
	(list 'lambda (cons accumulator parameters) (result_consumer_fold consumer accumulator body))
	_ (error "result scan requires an explicit native accumulator mapper"))))
(define compile_result_scan_plan (lambda (consumer arguments) (begin
	(define operator (car arguments))
	/* Batch acceptance already owns a compiled access schema at its caller. */
	(define compiled (equal? operator 'scan_order_batch_accept))
	(if (nil? consumer) (if compiled arguments (apply compile_scan_plan arguments)) (begin
		(define layout (if (equal? operator 'scan) '(6 7 8 false)
			(equal? operator 'scan_order) '(11 12 nil false)
			(equal? operator 'scan_order_batch_accept) '(12 13 nil false)
			(equal? operator 'scan_order_multi) '(13 14 nil true)
			(equal? operator 'scan_join_order) '(14 15 16 false)
			(error "unsupported final scan consumer operator")))
		(define rewritten (mapIndex arguments (lambda (index value)
			(if (equal? index (car layout))
				(if (nth layout 3) (cons 'list (map (scan_plan_list value)
					(lambda (mapper) (result_consumer_scan_mapper consumer mapper))))
					(result_consumer_scan_mapper consumer value))
				(equal? index (cadr layout)) (if compiled (scan_plan_initializer (result_consumer_initial consumer)) (result_consumer_initial consumer))
				(and (not (nil? (nth layout 2))) (equal? index (nth layout 2))) (result_consumer_combine consumer)
				value))))
		(if compiled rewritten (apply compile_scan_plan rewritten)))))))
/* Direct dictionary groups already own a serial accepted-row/window state.
Carry the final consumer's accumulator beside that ordinal, rather than a cell
or another scan over the input. HAVING and OFFSET/LIMIT still select SQL rows
before the output callback or its reducer can execute. */
(define result_consumer_grouped_rows (lambda (consumer grouped rowassoc having emit offset limit) (begin
	(define expressions (list rowassoc having emit offset limit consumer))
	(define state (result_consumer_temporary expressions 0))
	(define result (result_consumer_temporary (append expressions state) 0))
	(define reducer (list 'lambda (list state '__group_key '__group_payload)
		(list 'begin
			(list 'define '__accepted (list 'car state))
			(list 'define result (list 'cadr state))
			(list 'define 'rowassoc rowassoc)
			(list 'if having
				(list 'if (list '< '__accepted offset)
					(list 'list (list '+ '__accepted 1) result)
					(list 'if (list 'or (list 'equal? limit -1)
						(list '< (list '- '__accepted offset) limit))
						(list 'list (list '+ '__accepted 1) (result_consumer_fold consumer result emit))
						state))
				state))))
	(list 'cadr (list 'reduce_assoc grouped reducer
		(list 'list 0 (result_consumer_initialize consumer)))))))
