(settings "JITLog" true)
(print "start x+1")
(print ((jit (lambda (x) (+ x 1))) 4))
(print "done")
