(begin
  (define fa (jit (lambda (a b) a)))
  (define fb (jit (lambda (a b) b)))
  (print (list "a" (fa 3 4)))
  (print (list "b" (fb 3 4)))
)
