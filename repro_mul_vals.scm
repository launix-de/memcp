(settings "JITLog" false)
(print ((jit (lambda (a b) (* a b))) 1 7))
(print ((jit (lambda (a b) (* a b))) 2 9))
(print ((jit (lambda (a b) (* a b))) -3 2))
