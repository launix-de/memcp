/*
Copyright (C) 2023  Carl-Philip Hänsch

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

(define minigame_static '(
	"" '(200 "text/html" "<script type='text/javascript' src='game.js'></script>Have fun, play with it")
	"game.js" '(200 "text/javascript" "window.onload = function () {
		conn = new WebSocket('ws://' + document.location.host + '/minigame/ws');
		conn.onopen = function () {
			conn.send('hi from client');
		}
		conn.onmessage = function (msg) {
			console.log(msg);
			alert(msg.data);
		}
	}")
	'(404 "text/plain" "404 not found")
))

(define http_handler (begin
	(set old_handler http_handler)
	(lambda (req res) (begin
		/* hooked our additional paths to it */
		(match (req "path")
			(regex "^/minigame/(.*)$" url rest) (begin
				(if (equal? rest "ws") (begin
					(set msg ((res "websocket") (lambda (msg) (print "message: " msg))))
					(msg 1 "Hello World from server")
				) (match (minigame_static rest) '(status type content) (begin
					((res "header") "Content-Type" type)
					((res "status") status)
					((res "print") content)
				)))
			)
			/* default */
			(old_handler req res))
	))
))
