(print ((jit (lambda (x) (+ (strlen x) 1))) "abcd"))
(print ((jit (lambda (x) (+ (strlen x) 1))) "a"))
(print ((jit (lambda (x) (+ (strlen x) 1))) ""))
