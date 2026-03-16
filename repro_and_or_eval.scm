(begin
  (define fa (lambda (x) (and (> x 0) (< x 10))))
  (define fo (lambda (x) (or (< x 0) (> x 10))))
  (print (list "and15-eval" (fa 15)))
  (print (list "or15-eval" (fo 15)))
)
