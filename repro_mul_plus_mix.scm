(print ((jit (lambda (a b c) (+ (* a b) c))) 3 4 5))
(print ((jit (lambda (a b c d) (+ (* a b) (* c d)))) 2 3 4 5))
(print ((jit (lambda (a b) (- (* a a) (* b b)))) 5 3))
