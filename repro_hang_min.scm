(settings "JITLog" true)
(print ((jit (lambda (x) (* (+ x 1) 2))) 4))
