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

/* check any authenticated user (returns admin flag or false) */
(define dashboard_check_user (lambda (req) (begin
	(set pw (scan "system" "user" '("username") (lambda (username) (equal? username (req "username"))) '("password" "admin") (lambda (password admin) (list password admin)) (lambda (a b) b) nil))
	(if (and pw (equal? (car pw) (password (req "password")))) (car (cdr pw)) nil)
)))

/* send 401 with WWW-Authenticate header */
(define dashboard_send_401 (lambda (res) (begin
	((res "header") "Content-Type" "text/plain")
	((res "header") "WWW-Authenticate" "Basic realm=\"MemCP Dashboard\"")
	((res "status") 401)
	((res "print") "Unauthorized")
)))

/* list databases visible to a user: admin sees all, non-admin sees system.access entries */
(define dashboard_user_databases (lambda (username is_admin) (begin
	(if is_admin (show)
		(begin
			(define access_set (scan "system" "access"
				'("username" "database") (lambda (u db) (equal?? u username))
				'("database") (lambda (db) db)
				(lambda (acc db) (set_assoc acc db true))
				(list)))
			(filter (show) (lambda (db) (has? access_set db)))
		)
	)
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

/* check if user has access to a specific database */
(define dashboard_has_db_access (lambda (username is_admin dbname)
	(if is_admin true
		(> (scan "system" "access"
			'("username" "database") (lambda (u db) (and (equal?? u username) (equal?? db dbname)))
			'() (lambda () 1)
			+ 0) 0)
	)
))

/* helper: authenticate + check db access; returns is_admin or sends error */
(define dashboard_check_db (lambda (req res dbname body)
	(begin
		(define is_admin (dashboard_check_user req))
		(if (nil? is_admin) (dashboard_send_401 res)
			(if (dashboard_has_db_access (req "username") is_admin dbname) (body is_admin)
				(begin ((res "status") 403) ((res "header") "Content-Type" "text/plain") ((res "print") "Access denied"))
			)
		)
	)
))

/* helper: sum size_bytes across all shards of a table */
(define dashboard_table_size (lambda (db tbl)
	(reduce (show_shards db tbl) (lambda (acc shard) (+ acc (shard "size_bytes"))) 0)
))

/* helper: join list of JSON strings into a JSON array */
(define dashboard_json_array (lambda (items)
	(if (nil? items) "[]" (concat "[" (reduce items (lambda (a b) (concat a "," b))) "]"))
))

/* helper: send JSON response with proper Content-Type */
(define dashboard_send_json (lambda (res body) (begin
	((res "header") "Content-Type" "application/json")
	((res "print") body)
)))

/* helper: build JSON for a single user entry (takes username string) */
(define dashboard_build_user_json (lambda (uname) (begin
	(set is_adm (scan "system" "user" '("username") (lambda (u) (equal? u uname)) '("admin") (lambda (a) a) (lambda (a b) b) false))
	/* get database access for non-admins */
	(set dbs_csv (if is_adm ""
		(scan "system" "access" '("username") (lambda (u) (equal?? u uname))
			'("database") (lambda (db) (json_encode db))
			(lambda (acc db) (if (equal? acc "") db (concat acc "," db))) "")))
	(concat "{\"username\":" (json_encode uname)
		",\"admin\":" (if is_adm "true" "false")
		",\"databases\":[" dbs_csv "]}")
)))

/* hook into http_handler */
(define http_handler (begin
	(set old_handler http_handler)
	/* register Dashboard in service registry */
	(if (not (nil? service_registry)) (begin
		(service_registry "Dashboard" (list (arg "api-port" (env "PORT" "4321")) "/dashboard" "GET, WebSocket"))
	))
	(lambda (req res) (begin
		(match (req "path")
			/* API: list running services */
			"/dashboard/api/services" (begin
				(if (nil? (dashboard_check_user req)) (dashboard_send_401 res) (begin
					(if (not (nil? service_registry)) (begin
						(define svc_keys (service_registry))
						(define items (map svc_keys (lambda (name) (begin
							(define val (service_registry name))
							(json_encode_assoc (list "name" name "port" (car val) "route" (car (cdr val)) "protocols" (car (cdr (cdr val)))))
						))))
						(dashboard_send_json res (dashboard_json_array items))
					) (dashboard_send_json res "[]"))
				))
			)
			/* API: list databases (admin: all, non-admin: filtered by system.access) */
			"/dashboard/api/databases" (begin
				(define is_admin (dashboard_check_user req))
				(if (nil? is_admin) (dashboard_send_401 res) (begin
					(define dbs (dashboard_user_databases (req "username") is_admin))
					(define items (map dbs (lambda (db) (begin
						(define tables (show db))
						(define table_count (if (nil? tables) 0 (count tables)))
						(define total_size (if (nil? tables) 0
							(reduce (map tables (lambda (tbl) (dashboard_table_size db tbl))) (lambda (a b) (+ a b)) 0)))
						(json_encode_assoc (list "name" db "tables" table_count "size_bytes" total_size))
					))))
					(dashboard_send_json res (dashboard_json_array items))
				))
			)
			/* API: read all settings */
			"/dashboard/api/settings" (begin
				(if (dashboard_check_admin req) (begin
					(if (equal? (req "method") "POST") (begin
						/* write a single setting: body is JSON {"key":"...", "value":...} */
						(define body (json_decode ((req "body"))))
						(settings (body "key") (body "value"))
						(dashboard_send_json res "{\"ok\":true}")
					) (begin
							/* read all settings as assoc object */
							(dashboard_send_json res (json_encode_assoc (settings)))
					))
				) (dashboard_send_401 res))
			)
			/* API: execute admin SQL (ALTER, DROP, etc.) */
			"/dashboard/api/sql" (begin
				(if (dashboard_check_admin req) (begin
					(define body (json_decode ((req "body"))))
					(define db (body "database"))
					(define sql (body "sql"))
					(eval (parse_sql db sql (lambda (schema table write) true)))
					(dashboard_send_json res "{\"ok\":true}")
				) (dashboard_send_401 res))
			)
			/* API: read-only query, streams NDJSON (same auth realm as dashboard) */
			"/dashboard/api/query" (begin
				(define is_admin (dashboard_check_user req))
				(if (nil? is_admin) (dashboard_send_401 res) (begin
					(define body (json_decode ((req "body"))))
					(define db (body "database"))
					(define sql (body "sql"))
					(define policy (if is_admin (lambda (schema table write) (not write)) (sql_policy (req "username"))))
					((res "header") "Content-Type" "application/x-ndjson")
					(define resultrow (res "jsonl"))
					(eval (parse_sql db sql policy))
				))
			)
			/* API: shard column detail with compression types */
			(regex "^/dashboard/api/db/([^/]+)/([^/]+)/shard/([0-9]+)$" _ dbname tblname shardidx) (begin
				(dashboard_check_db req res dbname (lambda (is_admin) (begin
					(define sidx (simplify shardidx))
					(define cols (show_shard_columns dbname tblname sidx))
					(define shards (show_shards dbname tblname))
					(define shard_info (if (nil? shards) nil (nth shards sidx)))
					(define main_count (if (nil? shard_info) 0 (shard_info "main_count")))
					(define delta_count (if (nil? shard_info) 0 (shard_info "delta")))
					(define deletions (if (nil? shard_info) 0 (shard_info "deletions")))
					(define shard_size (if (nil? shard_info) 0 (shard_info "size_bytes")))
					(define shard_state (if (nil? shard_info) "" (shard_info "state")))
					(define items (map cols (lambda (c)
						(json_encode_assoc (list
							"name" (c "name")
							"compression" (c "compression")
							"size_bytes" (c "size_bytes")
						))
					)))
					(define indexes (show_shard_indexes dbname tblname sidx))
					(define index_items (if (nil? indexes) "[]"
						(dashboard_json_array (map indexes (lambda (ix)
							(json_encode_assoc (list
								"cols" (ix "cols")
								"active" (ix "active")
								"savings" (ix "savings")
								"size_bytes" (ix "size_bytes")
							))
						)))
					))
					(dashboard_send_json res (concat
						"{\"main_count\":" (json_encode main_count)
						",\"delta_count\":" (json_encode delta_count)
						",\"deletions\":" (json_encode deletions)
						",\"size_bytes\":" (json_encode shard_size)
						",\"state\":" (json_encode shard_state)
						",\"columns\":" (dashboard_json_array items)
						",\"indexes\":" index_items "}"))
				)))
			)
			/* API: table detail with columns, shards, meta */
			(regex "^/dashboard/api/db/([^/]+)/([^/]+)$" _ dbname tblname) (begin
				(dashboard_check_db req res dbname (lambda (is_admin) (begin
					(define cols (show dbname tblname))
					(define shards (show_shards dbname tblname))
					(define meta (show dbname tblname "meta"))
					(define col_items (map cols (lambda (col)
						(json_encode_assoc (list
							"name" (col "Field")
							"type" (col "Type")
							"nullable" (col "Null")
							"key" (col "Key")
						))
					)))
					(define shard_items (map shards (lambda (s)
						(json_encode_assoc (list
							"shard" (s "shard")
							"state" (s "state")
							"main_count" (s "main_count")
							"delta" (s "delta")
							"deletions" (s "deletions")
							"size_bytes" (s "size_bytes")
						))
					)))
					(define raw_uniques (meta "Unique"))
					(define uniques (if (nil? raw_uniques) nil (filter raw_uniques (lambda (u) (not (nil? u))))))
					(define unique_items (if (or (nil? uniques) (equal? (count uniques) 0)) "[]"
						(dashboard_json_array (map uniques (lambda (u)
							(concat "{\"id\":" (json_encode (u "Id")) ",\"cols\":" (json_encode (u "Cols")) "}")
						)))
					))
					/* partition schema */
					(define raw_partitions (meta "Partitions"))
					(define partitions (if (nil? raw_partitions) nil (filter raw_partitions (lambda (p) (not (nil? p))))))
					(define partition_items (if (or (nil? partitions) (equal? (count partitions) 0)) "[]"
						(dashboard_json_array (map partitions (lambda (p)
							(concat "{\"column\":" (json_encode (p "Column")) ",\"n\":" (json_encode (p "NumPartitions")) "}")
						)))
					))
					/* build JSON manually to nest arrays inside object */
					(dashboard_send_json res (concat
						"{\"columns\":" (dashboard_json_array col_items)
						",\"shards\":" (dashboard_json_array shard_items)
						",\"meta\":{\"engine\":" (json_encode (meta "Engine"))
						",\"collation\":" (json_encode (meta "Collation"))
						",\"uniques\":" unique_items
						",\"partitions\":" partition_items "}}"
					))
				)))
			)
			(regex "^/dashboard/api/db/([^/]+)$" _ dbname) (begin
				(dashboard_check_db req res dbname (lambda (is_admin) (begin
					(define tables (show dbname))
					(define items (if (nil? tables) nil (map tables (lambda (tbl) (begin
						(define meta (show dbname tbl "meta"))
						(define cols (show dbname tbl))
						(define shards (show_shards dbname tbl))
						(define col_count (if (nil? cols) 0 (count cols)))
						(define shard_count (if (nil? shards) 0 (count shards)))
						(define total_size (dashboard_table_size dbname tbl))
						(define all_cold (if (nil? shards) true
							(reduce shards (lambda (acc s) (and acc (equal? (s "state") "nil"))) true)))
						(define row_count (if all_cold nil
							(if (nil? shards) 0
								(reduce shards (lambda (acc s) (+ acc (+ (s "main_count") (s "delta")) (- 0 (s "deletions")))) 0))))
						(json_encode_assoc (list
							"name" tbl
							"engine" (meta "Engine")
							"columns" col_count
							"shards" shard_count
							"rows" row_count
							"size_bytes" total_size
						))
					)))))
					(dashboard_send_json res (dashboard_json_array items))
				)))
			)
			/* API: list all users with their database access (admin only) */
			"/dashboard/api/users" (begin
				(if (dashboard_check_admin req) (begin
					/* build JSON per user directly in scan reduce (avoids multi-param lambda optimizer bug) */
					(set json_str (scan "system" "user" '() (lambda () true) '("username")
						(lambda (u) (dashboard_build_user_json u))
						(lambda (acc item) (if (equal? acc "") item (concat acc "," item)))
						""))
					(dashboard_send_json res (concat "[" json_str "]"))
				) (dashboard_send_401 res))
			)
			/* API: check current user's admin status */
			"/dashboard/api/whoami" (begin
				(define is_admin (dashboard_check_user req))
				(if (nil? is_admin)
					(dashboard_send_401 res)
					(begin
						(define default_pw (if is_admin
							(scan "system" "user" '("username") (lambda (username) (equal? username "root")) '("password") (lambda (pw) (equal? pw (password "admin"))) (lambda (a b) b) false)
							false))
						(dashboard_send_json res (concat
							"{\"admin\":" (if is_admin "true" "false")
							",\"username\":" (json_encode (req "username"))
							(if is_admin (concat ",\"default_password\":" (if default_pw "true" "false")) "")
							"}"))
					)
				)
			)
			"/dashboard" (begin
				(if (nil? (dashboard_check_user req))
					(dashboard_send_401 res)
					(begin
						((res "header") "Content-Type" "text/html; charset=utf-8")
						((res "print") (readfile "assets/dashboard.html"))
					)
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

/* Metrics tracing: periodically insert rows into system_statistic.perf_metrics */
(if (not (has? (show "system_statistic") "perf_metrics")) (begin
	(print "creating table system_statistic.perf_metrics")
	(eval (parse_sql "system_statistic" "CREATE TABLE perf_metrics(time text, cpu float, mem_available bigint, mem_total bigint, shard_memory bigint, shard_budget bigint, connections int, max_connections int, rps float) ENGINE=SLOPPY" (lambda (schema table write) true)))
))

/* self-scheduling tracing loop via setTimeout */
(define metrics_trace_tick (lambda () (begin
	(if (settings "MetricsTracing") (begin
		(set cs (cache_stat))
		(insert "system_statistic" "perf_metrics"
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
