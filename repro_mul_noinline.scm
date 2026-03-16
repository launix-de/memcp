(settings "JITLog" false)
(define mymul (lambda (x y) (* x y)))
(print ((jit (lambda (a b) (mymul a b))) 1 7))
(print ((jit (lambda (a b) (mymul a b))) 3 4))
