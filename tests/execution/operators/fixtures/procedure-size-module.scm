/* Copyright (C) 2026 Carl-Philip Hänsch
 * SPDX-License-Identifier: GPL-3.0-or-later */
(define size_fixture_named (lambda (value) (begin
 (set value (+ value 2))
 (eval (quote value)))))
(define size_fixture_parser (lambda (input)
 ((parser (define matched (regex "[a-z]+" true)) matched) input)))
(define size_fixture_make (lambda (native payload)
 (lambda (value) (begin
  (cluster_object_status true)
  (+ (native "value") (payload "value") (eval (quote value)))))))
(define size_fixture_quoted (lambda (_ignored) (quote ("value" "short"))))
