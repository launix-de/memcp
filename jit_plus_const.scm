(print ((jit (lambda () (+ 3 4)))))
(print ((jit (lambda (a b) (+ a b))) 3 4))
(print ((jit (lambda (a b) (- a b))) 10 3))
