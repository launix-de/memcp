(settings "JITLog" false)
(print ((jit (lambda () (* 2 3)))))
(print ((jit (lambda () (* 3 4)))))
(print ((jit (lambda () (* 2 3 4)))))
