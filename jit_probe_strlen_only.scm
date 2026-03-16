(settings "JITLog" true)
(print ((jit (lambda (x) (strlen x))) "abcd"))
