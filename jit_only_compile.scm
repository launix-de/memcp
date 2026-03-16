(settings "JITLog" true)
(define f (jit (lambda (x) (* (+ x 1) 2))))
(print "compiled")
