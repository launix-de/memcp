(settings "JITLog" false)
(print ((jit (lambda (x) (* x 2.0))) 5))
(print ((jit (lambda (x) (* x 2))) 5))
