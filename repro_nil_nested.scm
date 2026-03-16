(settings "JITLog" true)
(define f (jit (lambda (x) (nil? (nil? x)))))
(print (f nil))
(print (f 42))
