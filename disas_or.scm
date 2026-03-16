(begin
  (settings "JITLog" true)
  (define f (jit (lambda (x) (or (< x 0) (> x 10)))))
  (f 15)
)
