(begin
  (define flt (jit (lambda (x) (< x 10))))
  (print (list "<5" (flt 5)))
  (print (list "<15" (flt 15)))
)
