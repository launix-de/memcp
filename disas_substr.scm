(begin
  (settings "JITLog" true)
  (define f (jit (lambda (s i l) (substr s i l))))
  (f "abcdef" 0 3)
)
