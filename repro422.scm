(define jit_fallback_desc (jit (lambda () (now))))
(print (jit? jit_fallback_desc))
(print (number? (jit_fallback_desc)))
