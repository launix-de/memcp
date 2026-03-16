(print ((jit (lambda (x) (* (+ x 1) (+ x 2)))) 3))
(print ((jit (lambda (x) (* (+ x 1) 2))) 4))
(print ((jit (lambda (a b) (* a b))) 1 7))
(print ((jit (lambda (a b c) (+ (* a b) c))) 3 4 5))
