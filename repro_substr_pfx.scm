(begin
  (settings "JITLog" false)
  (define f3 (jit (lambda (s i l) (substr s i l))))
  (print (f3 "abcdef" 0 3))
)
