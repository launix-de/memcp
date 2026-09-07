/*
Copyright (C) 2026  Carl-Philip Haensch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

/* Persist source rather than runtime closures. Each enabled source must
evaluate to a one-argument callback. Runtime dispatch never reads this table. */
(if (has? (show "system") "storage_failure_hooks") true (begin
	(print "creating table system.storage_failure_hooks")
	(eval (parse_sql "system" "CREATE TABLE storage_failure_hooks(name text, enabled boolean, classes text, cooldown_seconds int, source text, UNIQUE KEY uniq_name (name)) ENGINE=SAFE" (lambda (schema tblname write) true)))
	(insert (table "system" "storage_failure_hooks")
		'("name" "enabled" "classes" "cooldown_seconds" "source")
		(list
			(list "syslog-process" false "*" 300
				"(lambda (failure) (process_run \"logger\" (list \"-t\" \"memcp-storage\" (serialize failure))))")
			(list "open-outage-page" false "ambiguous_commit,io" 300
				"(lambda (failure) (process_run \"xdg-open\" (list \"file:///etc/memcp/storage-failure.html\")))")))
))

(define storage_failure_hook_rows (lambda ()
	(scan nil (table "system" "storage_failure_hooks") '(369435906932736) '()
		'() (lambda () true)
		'("name" "enabled" "classes" "cooldown_seconds" "source")
		(lambda (rows name enabled classes cooldown_seconds source)
			(cons (list "name" name "enabled" enabled "classes" classes
				"cooldown_seconds" cooldown_seconds "source" source) rows))
		'() (lambda (a b) (merge (list a b)))))
)

(define storage_failure_hook_compile (lambda (name source)
	(eval (scheme source (concat "storage-failure-hook:" name))))
)

(define storage_failure_hook_validate_classes (lambda (classes)
	(match classes
		nil true
		(cons class rest) (begin
			(if (or (equal? class "") (> (count (split class ",")) 1))
				(error "storage failure hook classes must be non-empty names without commas")
				true)
			(storage_failure_hook_validate_classes rest)))
))

(define storage_failure_hook_activate (lambda (name enabled classes cooldown_seconds source) (begin
	(if enabled
		(register_storage_failure_hook name classes cooldown_seconds
			(storage_failure_hook_compile name source))
		(unregister_storage_failure_hook name))
	true
)))

(define storage_failure_hook_activate_rows (lambda (rows)
	(match rows
		nil true
		(cons row rest) (begin
			(try (lambda ()
				(storage_failure_hook_activate (row "name") (row "enabled")
					(split (row "classes") ",") (row "cooldown_seconds") (row "source")))
				(lambda (e) (print "storage failure hook " (row "name") " was not activated: " e)))
			(storage_failure_hook_activate_rows rest)))
))

(define reload_storage_failure_hooks (lambda () (begin
	(clear_storage_failure_hooks)
	(storage_failure_hook_activate_rows (storage_failure_hook_rows))
)))

(define storage_failure_hook_save (lambda (name enabled classes cooldown_seconds source) (begin
	(if (equal? name "") (error "storage failure hook name must not be empty") true)
	(if (equal? (count classes) 0) (error "storage failure hook must select at least one class") true)
	(storage_failure_hook_validate_classes classes)
	(if (< cooldown_seconds 0) (error "storage failure hook cooldown must not be negative") true)
	/* Compile and validate through the native registration before persisting so
	an enabled broken hook never enters the startup catalog. */
	(define callback (if enabled (storage_failure_hook_compile name source) nil))
	(define classes_text (reduce classes (lambda (a b) (concat a "," b))))
	(if enabled
		(register_storage_failure_hook name classes cooldown_seconds callback)
		true)
	(define updated (scan nil (table "system" "storage_failure_hooks") '(369435906932736) '()
		'("name") (lambda (existing_name) (equal? existing_name name))
		'("$update") (lambda (count $update) (begin
			($update (list "enabled" enabled "classes" classes_text
				"cooldown_seconds" cooldown_seconds "source" source))
			(+ count 1)))
		0 +))
	(if (equal? updated 0)
		(insert (table "system" "storage_failure_hooks")
			'("name" "enabled" "classes" "cooldown_seconds" "source")
			(list (list name enabled classes_text cooldown_seconds source)))
		true)
	(if enabled true (unregister_storage_failure_hook name))
	true
)))

(define storage_failure_hook_delete (lambda (name) (begin
	(scan nil (table "system" "storage_failure_hooks") '(369435906932736) '()
		'("name") (lambda (existing_name) (equal? existing_name name))
		'("$update") (lambda (count $update) (begin ($update) (+ count 1)))
		0 +)
	(unregister_storage_failure_hook name)
	true
)))

(reload_storage_failure_hooks)
