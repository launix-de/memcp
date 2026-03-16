(settings "JITLog" true)
(print ((jit (lambda (a b) (< a b))) 5 3))
