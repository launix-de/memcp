(begin
  (define f (jit (lambda (s i l) (+ i 0))))
  (print (list "i+0" (f "abcdef" 0 3)))
  (print (list "i+0b" (f "abcdef" 2 3)))
)
