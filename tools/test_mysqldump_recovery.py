#!/usr/bin/env python3

"""
Copyright (C) 2026 Carl-Philip Haensch

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
"""

from __future__ import annotations

import argparse
import os
from pathlib import Path
import secrets
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time


SOURCE_DATABASE = "memcp_recovery_source"

FIXTURE_SQL = r"""
CREATE DATABASE `memcp_recovery_source`;
USE `memcp_recovery_source`;

CREATE TABLE `fop_files` (
  `ID` INT PRIMARY KEY AUTO_INCREMENT,
  `filename` TEXT,
  `data` LONGBLOB,
  `uploaded_at` BIGINT,
  UNIQUE KEY `filename_unique` (`filename`(128))
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=INNODB;

CREATE TABLE `dokument` (
  `ID` INT PRIMARY KEY AUTO_INCREMENT,
  `file` INT,
  `kommentar` LONGTEXT,
  CONSTRAINT `dokument_file_fk` FOREIGN KEY (`file`)
    REFERENCES `fop_files` (`ID`) ON DELETE CASCADE
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=INNODB;

CREATE TABLE `fop_notification` (
  `user` BIGINT,
  `channel` TEXT,
  `dv` TEXT,
  `id` BIGINT,
  `date` BIGINT
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=INNODB;

INSERT INTO `fop_files` (`filename`, `data`, `uploaded_at`) VALUES
  ('Pruefung-Mueller.pdf', FROM_BASE64('AP8BgD8='), 1788172496),
  ('leer.txt', FROM_BASE64(''), 1788172497);

INSERT INTO `dokument` (`file`, `kommentar`) VALUES
  (1, 'Zeile 1\nZeile 2 mit Umlauten: äöüß'),
  (2, NULL);
"""

VERIFY_SQL = """
SELECT COUNT(*), SUM(`uploaded_at`), SUM(LENGTH(`data`)) FROM `fop_files`;
SELECT COUNT(*), SUM(`kommentar` IS NULL) FROM `dokument`;
SELECT TO_BASE64(`data`) FROM `fop_files` WHERE `ID` = 1;
SELECT COUNT(*) FROM `fop_notification`;
"""

EXPECTED_VERIFY_LINES = [
    "2\t3576344993\t5",
    "2\t1",
    "AP8BgD8=",
    "0",
]


class RecoveryError(RuntimeError):
    pass


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Start a disposable MemCP instance, dump an Omnestum-shaped fixture "
            "with mysqldump, restore it into MariaDB, and verify the result."
        )
    )
    parser.add_argument("--memcp-binary", default="./memcp")
    parser.add_argument("--app", default="lib/main.scm")
    parser.add_argument("--mariadb-client", default="mariadb")
    parser.add_argument("--mysqldump", default="mysqldump")
    parser.add_argument("--mariadb-user", default="root")
    parser.add_argument("--mariadb-host")
    parser.add_argument("--mariadb-port", type=int)
    parser.add_argument("--mariadb-socket")
    parser.add_argument(
        "--quiesced-empty-table-workaround",
        action="store_true",
        help=(
            "permit the documented empty-table workaround; source writes must "
            "remain stopped from empty-table discovery until mysqldump exits"
        ),
    )
    return parser.parse_args()


def free_port() -> int:
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        return int(listener.getsockname()[1])


def require_command(command: str) -> None:
    if "/" in command:
        if not Path(command).is_file():
            raise RecoveryError(f"required executable does not exist: {command}")
    elif shutil.which(command) is None:
        raise RecoveryError(f"required command is not installed: {command}")


def run(
    command: list[str],
    *,
    input_text: str | None = None,
    stdout=None,
    env: dict[str, str] | None = None,
    check: bool = True,
) -> subprocess.CompletedProcess[str]:
    completed = subprocess.run(
        command,
        input=input_text,
        stdout=stdout if stdout is not None else subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        env=env,
        check=False,
    )
    if check and completed.returncode != 0:
        detail = completed.stderr.strip() or completed.stdout.strip()
        raise RecoveryError(
            f"command failed ({completed.returncode}): {' '.join(command)}\n{detail}"
        )
    return completed


def run_with_stdin_file(command: list[str], path: Path) -> None:
    with path.open("rb") as source:
        completed = subprocess.run(
            command,
            stdin=source,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
    if completed.returncode != 0:
        detail = (completed.stderr or completed.stdout).decode("utf-8", "replace").strip()
        raise RecoveryError(
            f"command failed ({completed.returncode}): {' '.join(command)}\n{detail}"
        )


def client_auth_file(path: Path, *, host: str | None, port: int | None,
                     socket_path: str | None, user: str, password: str) -> None:
    lines = ["[client]", f"user={user}"]
    if password:
        lines.append(f"password={password}")
    if socket_path:
        lines.extend(["protocol=socket", f"socket={socket_path}"])
    else:
        lines.extend(["protocol=tcp", f"host={host or '127.0.0.1'}"])
        if port is not None:
            lines.append(f"port={port}")
    path.write_text("\n".join(lines) + "\n")
    path.chmod(0o600)


def wait_for_memcp(client: str, auth_file: Path, process: subprocess.Popen[str],
                   server_log: Path) -> None:
    deadline = time.monotonic() + 60
    command = [
        client,
        f"--defaults-extra-file={auth_file}",
        "--password=admin",
        "--batch",
        "-e",
        "SELECT 1",
    ]
    while time.monotonic() < deadline:
        if process.poll() is not None:
            detail = server_log.read_text(errors="replace")[-4000:]
            raise RecoveryError(
                f"MemCP exited during startup with status {process.returncode}\n{detail}"
            )
        if run(command, check=False).returncode == 0:
            return
        time.sleep(0.2)
    detail = server_log.read_text(errors="replace")[-4000:]
    raise RecoveryError(
        "MemCP MySQL endpoint did not become ready within 60 seconds\n" + detail
    )


def mysql_lines(client: str, auth_file: Path, sql: str,
                database: str | None = None,
                password_option: str | None = None) -> list[str]:
    command = [
        client,
        f"--defaults-extra-file={auth_file}",
        "--batch",
        "--skip-column-names",
    ]
    if password_option:
        command.insert(2, password_option)
    if database:
        command.append(database)
    completed = run(command, input_text=sql)
    return completed.stdout.splitlines()


def dump_command(dump_tool: str, auth_file: Path, dump_path: Path,
                 ignored_empty_tables: list[str]) -> list[str]:
    command = [
        dump_tool,
        f"--defaults-extra-file={auth_file}",
        "--password=admin",
        "--skip-comments",
        "--skip-lock-tables",
        "--single-transaction",
        "--hex-blob",
        "--default-character-set=utf8mb4",
    ]
    command.extend(
        f"--ignore-table-data={SOURCE_DATABASE}.{table}"
        for table in ignored_empty_tables
    )
    command.extend(["--result-file", str(dump_path), SOURCE_DATABASE])
    return command


def find_empty_tables(client: str, auth_file: Path) -> list[str]:
    tables = mysql_lines(
        client, auth_file, "SHOW TABLES", SOURCE_DATABASE, "--password=admin"
    )
    empty = []
    for table in tables:
        escaped = table.replace("`", "``")
        count = mysql_lines(
            client,
            auth_file,
            f"SELECT COUNT(*) FROM `{escaped}`",
            SOURCE_DATABASE,
            "--password=admin",
        )
        if count == ["0"]:
            empty.append(table)
    return empty


def maria_client_args(args: argparse.Namespace, auth_file: Path) -> list[str]:
    return [args.mariadb_client, f"--defaults-extra-file={auth_file}"]


def verify_restore(args: argparse.Namespace, auth_file: Path,
                   database: str) -> None:
    actual = mysql_lines(args.mariadb_client, auth_file, VERIFY_SQL, database)
    if actual != EXPECTED_VERIFY_LINES:
        raise RecoveryError(
            "restored data differs from the MemCP fixture\n"
            f"expected: {EXPECTED_VERIFY_LINES}\nactual:   {actual}"
        )

    post_restore = mysql_lines(
        args.mariadb_client,
        auth_file,
        """
        INSERT INTO `fop_files` (`filename`, `data`, `uploaded_at`)
          VALUES ('nach-restore.txt', FROM_BASE64('AQI='), 1788172498);
        SELECT LAST_INSERT_ID();
        DELETE FROM `fop_files` WHERE `ID` = 1;
        SELECT COUNT(*) FROM `dokument` WHERE `file` = 1;
        """,
        database,
    )
    if post_restore != ["3", "0"]:
        raise RecoveryError(
            "AUTO_INCREMENT or restored foreign-key cascade is incorrect: "
            f"{post_restore}"
        )


def main() -> int:
    args = parse_args()
    require_command(args.memcp_binary)
    require_command(args.mariadb_client)
    require_command(args.mysqldump)
    if not Path(args.app).is_file():
        raise RecoveryError(f"Scheme application does not exist: {args.app}")

    destination_password = os.environ.get("MEMCP_RECOVERY_MARIADB_PASSWORD", "")
    target_database = f"memcp_recovery_{os.getpid()}_{secrets.token_hex(4)}"
    mysql_port = free_port()
    api_port = free_port()

    with tempfile.TemporaryDirectory(prefix="memcp-mysqldump-recovery-") as temp_name:
        temp = Path(temp_name)
        data_dir = temp / "data"
        data_dir.mkdir()
        source_auth = temp / "source.cnf"
        destination_auth = temp / "destination.cnf"
        dump_path = temp / "recovery.sql"
        server_log = temp / "memcp.log"
        client_auth_file(
            source_auth,
            host="127.0.0.1",
            port=mysql_port,
            socket_path=None,
            user="root",
            password="admin",
        )
        client_auth_file(
            destination_auth,
            host=args.mariadb_host,
            port=args.mariadb_port,
            socket_path=args.mariadb_socket,
            user=args.mariadb_user,
            password=destination_password,
        )

        with server_log.open("w") as log:
            process = subprocess.Popen(
                [
                    args.memcp_binary,
                    "-data",
                    str(data_dir),
                    f"--api-port={api_port}",
                    f"--mysql-port={mysql_port}",
                    "--mysql-socket=",
                    "--no-repl",
                    args.app,
                ],
                stdin=subprocess.DEVNULL,
                stdout=log,
                stderr=subprocess.STDOUT,
                text=True,
            )
        destination_created = False
        try:
            wait_for_memcp(args.mariadb_client, source_auth, process, server_log)
            mysql_lines(
                args.mariadb_client,
                source_auth,
                FIXTURE_SQL,
                password_option="--password=admin",
            )
            source_values = mysql_lines(
                args.mariadb_client,
                source_auth,
                VERIFY_SQL,
                SOURCE_DATABASE,
                "--password=admin",
            )
            if source_values != EXPECTED_VERIFY_LINES:
                raise RecoveryError(f"MemCP fixture verification failed: {source_values}")

            plain_dump = run(
                dump_command(args.mysqldump, source_auth, dump_path, []),
                check=False,
            )
            if plain_dump.returncode != 0:
                if not args.quiesced_empty_table_workaround:
                    detail = plain_dump.stderr.strip()
                    raise RecoveryError(
                        "plain mysqldump failed; production recovery gate is closed\n"
                        f"{detail}\n"
                        "If application writes are stopped for the complete backup "
                        "window, rerun with --quiesced-empty-table-workaround."
                    )
                empty_tables = find_empty_tables(args.mariadb_client, source_auth)
                if not empty_tables:
                    raise RecoveryError(
                        "mysqldump failed, but no empty table was found; refusing workaround"
                    )
                print(
                    "plain mysqldump rejected empty result metadata; applying the "
                    "write-quiesced workaround for: " + ", ".join(empty_tables)
                )
                run(dump_command(args.mysqldump, source_auth, dump_path, empty_tables))
            else:
                print("plain mysqldump completed successfully")

            destination = maria_client_args(args, destination_auth)
            run(destination, input_text=f"CREATE DATABASE `{target_database}` CHARACTER SET utf8mb4")
            destination_created = True
            run_with_stdin_file(destination + [target_database], dump_path)
            verify_restore(args, destination_auth, target_database)
            print(
                "PASS: MemCP mysqldump restored into MariaDB with matching rows, "
                "binary data, empty-table schema, AUTO_INCREMENT, and FK cascade"
            )
            return 0
        finally:
            if destination_created:
                run(
                    maria_client_args(args, destination_auth),
                    input_text=f"DROP DATABASE IF EXISTS `{target_database}`",
                    check=False,
                )
            if process.poll() is None:
                process.send_signal(signal.SIGTERM)
                try:
                    process.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except RecoveryError as error:
        print(f"FAIL: {error}", file=sys.stderr)
        raise SystemExit(1)
