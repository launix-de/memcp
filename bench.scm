(settings "TracePrint" true)
(define N 1000000)

; --- (+ x 1) ---
(define f_scm (lambda (x) (+ x 1)))
(define f_jit (jit (lambda (x) (+ x 1))))
(time (for '(0) (lambda (i) (< i N)) (lambda (i) (begin (f_scm i) (list (+ i 1))))) "scm (+ x 1)")
(time (for '(0) (lambda (i) (< i N)) (lambda (i) (begin (f_jit i) (list (+ i 1))))) "jit (+ x 1)")

; --- constant 42 ---
(define g_scm (lambda () 42))
(define g_jit (jit (lambda () 42)))
(time (for '(0) (lambda (i) (< i N)) (lambda (i) (begin (g_scm) (list (+ i 1))))) "scm const")
(time (for '(0) (lambda (i) (< i N)) (lambda (i) (begin (g_jit) (list (+ i 1))))) "jit const")

; --- identity ---
(define h_scm (lambda (a) a))
(define h_jit (jit (lambda (a) a)))
(time (for '(0) (lambda (i) (< i N)) (lambda (i) (begin (h_scm i) (list (+ i 1))))) "scm identity")
(time (for '(0) (lambda (i) (< i N)) (lambda (i) (begin (h_jit i) (list (+ i 1))))) "jit identity")

; --- polynomial x^2+2x+1 ---
(define p_scm (lambda (x) (+ (* x x) (* 2 x) 1)))
(define p_jit (jit (lambda (x) (+ (* x x) (* 2 x) 1))))
(time (for '(0) (lambda (i) (< i N)) (lambda (i) (begin (p_scm i) (list (+ i 1))))) "scm poly")
(time (for '(0) (lambda (i) (< i N)) (lambda (i) (begin (p_jit i) (list (+ i 1))))) "jit poly")
