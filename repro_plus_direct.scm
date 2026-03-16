(print ((jit (lambda (a b) (+ a b))) 1 7))
(print ((jit (lambda (a b) (+ a 1))) 3 4))
(print ((jit (lambda (a b) (+ b 1))) 3 4))
