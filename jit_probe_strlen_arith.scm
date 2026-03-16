(print ((jit (lambda (x) (- (strlen x) 0))) "abcd"))
(print ((jit (lambda (x) (+ (strlen x) 0))) "abcd"))
