(settings "JITLog" true)
(print ((jit (lambda (a) (+ (strlen a) 1))) "hello"))
(print ((jit (lambda (s i n) (substr s i n))) "abcdef" 1 3))
