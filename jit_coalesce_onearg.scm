(print ((jit (lambda (a) (coalesce a))) 99))
(print ((jit (lambda (a) (coalesce a))) nil))
(print ((jit (lambda (a) (coalesce a))) "abc"))
(exit)
