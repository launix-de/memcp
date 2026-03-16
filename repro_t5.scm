(settings "JITLog" true)
(print "start")
(print ((jit (lambda (x) (* (+ x 1) 2))) 4))
(print "done")
