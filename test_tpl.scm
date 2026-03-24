(import "memcp/lib/rdf.scm")
(import "lib/rdfop.scm")
(define w (lambda (fn cb) nil))

/* Test BEFORE any SPARQL */
(try (lambda () (begin
    (parse_rdfhp "rdf" "\n@PREFIX rdfop: <https://launix.de/rdfop/schema#> .\nSELECT ?h WHERE { <urn:uuid:test> rdfop:html ?h }\nBEGIN\n?><div class='t'><?rdf PRINT RAW ?h ?></div><?rdf\nEND" w)
    (print "BEFORE OK")
)) (lambda (e) (print "BEFORE FAIL: " e)))

/* Run a SPARQL query (defines resultrow) */
(createdatabase "rdf" true)
(createtable "rdf" "rdf" '('("column" "s" "text" '() '()) '("column" "p" "text" '() '()) '("column" "o" "text" '() '()) '("unique" "u" '("s" "p" "o"))) '() true)
(load_ttl "rdf" "main a Test .")
(define resultrow (lambda (o) nil))
(eval (parse_sparql "rdf" "SELECT ?s WHERE { ?s a ?t }"))

/* Test AFTER SPARQL */
(try (lambda () (begin
    (parse_rdfhp "rdf" "\n@PREFIX rdfop: <https://launix.de/rdfop/schema#> .\nSELECT ?h WHERE { <urn:uuid:test> rdfop:html ?h }\nBEGIN\n?><div class='t'><?rdf PRINT RAW ?h ?></div><?rdf\nEND" w)
    (print "AFTER OK")
)) (lambda (e) (print "AFTER FAIL: " e)))
