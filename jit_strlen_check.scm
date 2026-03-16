(print "s1=" ((jit (lambda (x) (strlen x))) "hello"))
(print "s2=" ((jit (lambda (x) (strlen x))) (string 1)))
(print "s3=" ((jit (lambda (x) (strlen x))) 123))
(exit)
