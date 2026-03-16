(print ((jit (lambda (x) (int? x))) 4))
(print ((jit (lambda (x) (number? x))) 4))
(print ((jit (lambda (x) x)) 4))
