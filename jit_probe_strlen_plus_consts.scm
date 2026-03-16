(print ((jit (lambda (x) (+ (strlen x) 1 1))) "abcd"))
(print ((jit (lambda (x) (+ (strlen x) 5))) "abcd"))
