;; warmup similar to test suite
(print ((jit (lambda (x) (+ x 1))) 4))
(print ((jit (lambda (a b) (+ a b))) 3 4))
(print ((jit (lambda (a b) (+ a b))) 1.5 2.5))
(print ((jit (lambda (a b c d) (+ (* a b) (* c d)))) 2 3 4 5))
;; target
(print ((jit (lambda (a) (+ (strlen a) 1))) "hello"))
(print ((jit (lambda (s i n) (substr s i n))) "abcdef" 1 3))
