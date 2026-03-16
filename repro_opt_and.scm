(begin
  (print (optimize (quote (lambda (x) (and (> x 0) (< x 10))))))
  (print (optimize (quote (lambda (x) (or (< x 0) (> x 10))))))
)
