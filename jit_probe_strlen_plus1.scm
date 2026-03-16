(settings "JITLog" true)
(print ((jit (lambda (x) (+ (strlen x) 1))) "abcd"))
