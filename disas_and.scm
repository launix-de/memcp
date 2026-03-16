(begin
  (settings "JITLog" true)
  (define f (jit (lambda (x) (and (> x 0) (< x 10)))))
  (f 15)
)
