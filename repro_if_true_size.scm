(settings "JITLog" true)
(define f (jit (lambda (x) (if true (+ x 1) (+ x 2)))))
(print (f 10))
