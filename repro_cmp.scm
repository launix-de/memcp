(begin
  (define fgt (jit (lambda (x) (> x 0))))
  (define flt (jit (lambda (x) (< x 10))))
  (print (list ">" (fgt 15)))
  (print (list "<" (flt 15)))
)
