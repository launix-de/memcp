(settings "JITLog" false)
(print ((jit (lambda (x) (<= x 0))) -1))
(print ((jit (lambda (x) (<= x 0))) 0))
(print ((jit (lambda (x) (<= x 0))) 1))
