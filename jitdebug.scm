(print ((jit (lambda (x) (+ x 1))) 4))
(print ((jit (lambda (x) (- x 3))) 10))
(print ((jit (lambda (a b) (+ a b))) 3 4))
(print ((jit (lambda (a b) (- a b))) 10 3))
