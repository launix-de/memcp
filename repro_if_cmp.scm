(begin
  (define f (jit (lambda (x) (if (< x 10) true false))))
  (print (list "if5" (f 5)))
  (print (list "if15" (f 15)))
)
