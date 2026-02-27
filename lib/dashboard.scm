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

/* Dashboard: admin-only live metrics via WebSocket */

/* check admin credentials against system.user table */
(define dashboard_check_admin (lambda (req) (begin
	(set pw (scan "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password" "admin") (lambda (password admin) (list password admin)) (lambda (a b) b) nil))
	(and pw (equal? (car pw) (password (req "password"))) (car (cdr pw)))
)))

/* send 401 with WWW-Authenticate header */
(define dashboard_send_401 (lambda (res) (begin
	((res "header") "Content-Type" "text/plain")
	((res "header") "WWW-Authenticate" "Basic realm=\"MemCP Dashboard\"")
	((res "status") 401)
	((res "print") "Unauthorized (admin required)")
)))

/* WebSocket push loop: send metrics JSON every 100ms */
(define dashboard_push (lambda (send) (begin
	(send 1 (json_encode_assoc (list
		"cpu" (cpu_usage)
		"mem_available" (available_memory)
		"mem_total" (total_memory)
		"shard" (cache_stat)
		"process_memory" (process_memory)
		"connections" (active_connections)
		"max_connections" (max_connections)
		"rps" (requests_per_second)
	)))
	(sleep 0.1)
	(dashboard_push send)
)))

/* hook into http_handler */
(define http_handler (begin
	(set old_handler http_handler)
	old_handler old_handler /* workaround for optimizer bug */
	(lambda (req res) (begin
		(match (req "path")
			"/dashboard" (begin
				(if (dashboard_check_admin req)
					(begin
						((res "header") "Content-Type" "text/html; charset=utf-8")
						((res "print") (readfile "assets/dashboard.html"))
					)
					(dashboard_send_401 res)
				)
			)
			"/dashboard/logo.svg" (begin
				((res "header") "Content-Type" "image/svg+xml")
				((res "print") (readfile "assets/memcp-logo.svg"))
			)
			"/ws/dashboard" (begin
				(if (dashboard_check_admin req)
					(begin
						(set send ((res "websocket") (lambda (msg) nil)))
						(dashboard_push send)
					)
					(dashboard_send_401 res)
				)
			)
			/* default: fall through to previous handler */
			(old_handler req res))
	))
))

/* Metrics tracing: periodically insert rows into system.perf_metrics */
(if (not (has? (show "system") "perf_metrics")) (begin
	(print "creating table system.perf_metrics")
	(eval (parse_sql "system" "CREATE TABLE perf_metrics(time text, cpu float, mem_available bigint, mem_total bigint, shard_memory bigint, shard_budget bigint, connections int, max_connections int, rps float) ENGINE=SLOPPY" (lambda (schema table write) true)))
))

/* self-scheduling tracing loop via setTimeout */
(define metrics_trace_tick (lambda () (begin
	(if (settings "MetricsTracing") (begin
		(set cs (cache_stat))
		(insert "system" "perf_metrics"
			'("time" "cpu" "mem_available" "mem_total" "shard_memory" "shard_budget" "connections" "max_connections" "rps")
			(list (list
				(now)
				(cpu_usage)
				(available_memory)
				(total_memory)
				(cs "persisted_memory")
				(cs "persisted_budget")
				(active_connections)
				(max_connections)
				(requests_per_second)
			))
		)
	))
	/* reschedule: use configured interval (default 60s) */
	(set interval (settings "MetricsTracingInterval"))
	(if (<= interval 0) (set interval 60))
	(setTimeout metrics_trace_tick (* interval 1000))
)))

/* start the tracing loop */
(setTimeout metrics_trace_tick 60000)
