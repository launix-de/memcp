(print "jit strlen+1" ((jit (lambda (a) (+ (strlen a) 1))) "hello"))
(print "int strlen+1" (+ (strlen "hello") 1))
(print "jit substr" ((jit (lambda (s i n) (substr s i n))) "abcdef" 1 3))
(print "int substr" (substr "abcdef" 1 3))
