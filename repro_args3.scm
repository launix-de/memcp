(begin
  (define fi (jit (lambda (s i l) i)))
  (define fl (jit (lambda (s i l) l)))
  (print (list "i" (fi "abcdef" 0 3)))
  (print (list "l" (fl "abcdef" 0 3)))
)
