(print "mul1" ((jit (lambda (x) (* x 2))) 5))
(print "mul2" ((jit (lambda (a b c) (* (* a 2) (* b c))) 3 4 5))
