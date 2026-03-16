(print ((jit (lambda (a) a)) 5))
(print ((jit (lambda (a b) b)) 2 7))
(print ((jit (lambda (a b) a)) 2 7))
