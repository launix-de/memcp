(begin
  (define fa (jit (lambda (x) (and (> x 0) (< x 10)))))
  (define fo (jit (lambda (x) (or (< x 0) (> x 10)))))
  (print (list "and15" (fa 15)))
  (print (list "or15" (fo 15)))
)
