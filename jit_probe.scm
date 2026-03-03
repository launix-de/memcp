(print ((jit (lambda (x) (int? x))) 4))
(print ((jit (lambda (x) x)) 4))
(print ((jit (lambda (x) (+ x 1))) 4))
(print ((jit (lambda (a b) (+ a b))) 3 4))
