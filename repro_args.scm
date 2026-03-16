(print ((jit (lambda (a b) a)) 3 4))
(print ((jit (lambda (a b) b)) 3 4))
(print ((jit (lambda (a b) (+ b 0))) 3 4))
(print ((jit (lambda (a b) (+ 0 b))) 3 4))
