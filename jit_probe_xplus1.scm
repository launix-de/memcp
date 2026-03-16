(settings "JITLog" false)
(define f (jit (lambda (x) (+ x 1))))
(print (f 4))
