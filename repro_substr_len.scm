(begin
  (define f3 (jit (lambda (s i l) (substr s i l))))
  (define r (f3 "abcdef" 0 3))
  (print (strlen r))
)
