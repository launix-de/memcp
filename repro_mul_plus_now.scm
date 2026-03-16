(settings "JITLog" false)
(print ((jit (lambda (a b) (* a b))) 3 4))
(print ((jit (lambda (x) (* (+ x 1) 2))) 4))
(print ((jit (lambda (x) (+ (* x 2) 1))) 4))
