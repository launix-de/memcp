(print ((jit (lambda (a b c) (+ a c))) 3 4 5))
(print ((jit (lambda (a b c) (+ b c))) 3 4 5))
(print ((jit (lambda (a b c) (+ a b c))) 3 4 5))
