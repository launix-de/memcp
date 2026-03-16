(begin
  (settings "JITLog" false)
  (define f (jit (lambda (a b) (+ a b))))
  (print (list "3+4" (f 3 4)))
  (print (list "3+5" (f 3 5)))
  (print (list "4+3" (f 4 3)))
  (print (list "10+1" (f 10 1)))
  (print (list "1+10" (f 1 10)))
)
