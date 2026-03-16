(settings "JITLog" true)
(print "jit x+1 => " ((jit (lambda (x) (+ x 1))) 4))
(print "jit x*2 => " ((jit (lambda (x) (* x 2))) 5))
