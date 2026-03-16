(settings "JITLog" false)
(print ((jit (lambda (x c) (* x c))) 5 2))
(print ((jit (lambda (x c) (* c x))) 2 5))
