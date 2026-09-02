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
import json
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

CREATE TABLE `recovery_audit` (
  `file_id` INT,
  `event_name` VARCHAR(32)
) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=INNODB;

CREATE TABLE `recovery_parent` (`id` INT PRIMARY KEY) ENGINE=INNODB;
CREATE TABLE `recovery_child` (
  `id` INT PRIMARY KEY,
  `parent_id` INT,
  CONSTRAINT `recovery_child_parent_fk` FOREIGN KEY (`parent_id`)
    REFERENCES `recovery_parent` (`id`) ON DELETE CASCADE
) ENGINE=INNODB;
CREATE TABLE `recovery_unique` (
  `value` VARCHAR(64),
  UNIQUE KEY `recovery_value_unique` (`value`)
) ENGINE=INNODB;

CREATE TABLE `recovery_keyword_columns` (
  `method` TEXT,
  `parameter` TEXT,
  `user` BIGINT,
  `lock` BIGINT,
  `start` BIGINT,
  `ID` INT PRIMARY KEY AUTO_INCREMENT
) ENGINE=INNODB;

CREATE TABLE `recovery_statement_columns` (
  `database` TEXT,
  `name` TEXT,
  `dialect` TEXT,
  `sql` TEXT,
  `ir` TEXT
) ENGINE=INNODB;

CREATE TABLE `recovery_error_columns` (
  `datetime` TEXT,
  `database` TEXT,
  `user` TEXT,
  `query` TEXT,
  `error` TEXT
) ENGINE=INNODB;

INSERT INTO `fop_files` (`filename`, `data`, `uploaded_at`) VALUES
  ('Pruefung-Mueller.pdf', FROM_BASE64('AP8BgD8='), 1788172496),
  ('leer.txt', FROM_BASE64(''), 1788172497);

INSERT INTO `dokument` (`file`, `kommentar`) VALUES
  (1, 'Zeile 1\nZeile 2 mit Umlauten: äöüß'),
  (2, NULL);

INSERT INTO `recovery_parent` VALUES (1);
INSERT INTO `recovery_child` VALUES (1, 1);
INSERT INTO `recovery_unique` VALUES ('only-once');
INSERT INTO `recovery_keyword_columns`
  (`method`, `parameter`, `user`, `lock`, `start`) VALUES
  ('dispatch', '[1]', 7, NULL, 1788260000);
INSERT INTO `recovery_statement_columns` VALUES
  ('fixture', 'sample_view', 'mysql', 'SELECT 1', '(resultrow 1)');
INSERT INTO `recovery_error_columns` VALUES
  ('2026-09-02 10:00:00', 'fixture', 'backup', 'SELECT 1', 'synthetic');

CREATE TRIGGER `recovery_fop_files_ai` AFTER INSERT ON `fop_files`
  FOR EACH ROW INSERT INTO `recovery_audit` (`file_id`, `event_name`)
  VALUES (NEW.`ID`, 'insert');
"""

VERIFY_SQL = """
SELECT COUNT(*), SUM(`uploaded_at`), SUM(LENGTH(`data`)) FROM `fop_files`;
SELECT COUNT(*), SUM(`kommentar` IS NULL) FROM `dokument`;
SELECT TO_BASE64(`data`) FROM `fop_files` WHERE `ID` = 1;
SELECT COUNT(*) FROM `fop_notification`;
SELECT COUNT(*) FROM `recovery_audit`;
SELECT COUNT(*), SUM(`user`), SUM(`lock` IS NULL) FROM `recovery_keyword_columns`;
SELECT COUNT(*), MIN(`database`), MIN(`sql`) FROM `recovery_statement_columns`;
SELECT COUNT(*), MIN(`database`), MIN(`query`) FROM `recovery_error_columns`;
"""

EXPECTED_VERIFY_LINES = [
    "2\t3576344993\t5",
    "2\t1",
    "AP8BgD8=",
    "0",
    "0",
    "1\t7\t1",
    "1\tfixture\tSELECT 1",
    "1\tfixture\tSELECT 1",
]

FIXTURE_COLUMNS = {
    "fop_files": ["ID", "filename", "data", "uploaded_at"],
    "dokument": ["ID", "file", "kommentar"],
    "fop_notification": ["user", "channel", "dv", "id", "date"],
    "recovery_audit": ["file_id", "event_name"],
    "recovery_parent": ["id"],
    "recovery_child": ["id", "parent_id"],
    "recovery_unique": ["value"],
    "recovery_keyword_columns": ["method", "parameter", "user", "lock", "start", "ID"],
    "recovery_statement_columns": ["database", "name", "dialect", "sql", "ir"],
    "recovery_error_columns": ["datetime", "database", "user", "query", "error"],
}


class RecoveryError(RuntimeError):
    pass


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Start a disposable MemCP instance, dump an application-shaped fixture "
            "and MemCP system databases as a restricted user with mysqldump, restore "
            "the application dump into MariaDB and a fresh MemCP instance, and verify "
            "both results."
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
        "--application-schema-json",
        type=Path,
        help="create and validate every table described by an application sql_schema.json",
    )
    return parser.parse_args()


def quote_identifier(value: str) -> str:
    return "`" + value.replace("`", "``") + "`"


def application_fixture(schema_path: Path) -> tuple[str, dict[str, list[str]]]:
    try:
        schema = json.loads(schema_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise RecoveryError(f"cannot read application schema {schema_path}: {error}") from error
    if not isinstance(schema, dict) or not schema:
        raise RecoveryError(f"application schema is empty or invalid: {schema_path}")

    statements = [
        f"CREATE DATABASE {quote_identifier(SOURCE_DATABASE)}",
        f"USE {quote_identifier(SOURCE_DATABASE)}",
    ]
    expected_columns: dict[str, list[str]] = {}
    for table_name, table_spec in schema.items():
        columns = table_spec.get("columns", {})
        if not isinstance(columns, dict) or not columns:
            raise RecoveryError(f"application table {table_name} has no columns")
        expected_columns[table_name] = list(columns)
        definitions = []
        for column_name, column_spec in columns.items():
            definition = f"{quote_identifier(column_name)} {column_spec['type']}"
            if column_spec.get("primary"):
                definition += " PRIMARY KEY AUTO_INCREMENT"
            definitions.append(definition)
        for unique_name, unique_spec in table_spec.get("uniques", {}).items():
            unique_columns = ", ".join(
                quote_identifier(column) for column in unique_spec.values()
            )
            definitions.append(
                f"UNIQUE KEY {quote_identifier(unique_name)} ({unique_columns})"
            )
        statements.append(
            f"CREATE TABLE {quote_identifier(table_name)} ({', '.join(definitions)}) "
            "CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=INNODB"
        )

    for table_name, table_spec in schema.items():
        for index_name, index_spec in table_spec.get("indexes", {}).items():
            index_columns = ", ".join(
                quote_identifier(column) for column in index_spec.values()
            )
            statements.append(
                f"CREATE INDEX {quote_identifier(index_name)} ON "
                f"{quote_identifier(table_name)} ({index_columns})"
            )
        for column_name, column_spec in table_spec.get("columns", {}).items():
            for reference_name, reference in column_spec.get("references", {}).items():
                delete_action = "CASCADE" if reference.get("cascades") else "RESTRICT"
                statements.append(
                    f"ALTER TABLE {quote_identifier(table_name)} ADD CONSTRAINT "
                    f"{quote_identifier(reference_name)} FOREIGN KEY "
                    f"({quote_identifier(column_name)}) REFERENCES "
                    f"{quote_identifier(reference['table'])} "
                    f"({quote_identifier(reference['column'])}) "
                    f"ON UPDATE CASCADE ON DELETE {delete_action}"
                )

    statements.extend(
        [
            "CREATE TABLE `recovery_audit` (`file_id` INT, `event_name` VARCHAR(32)) "
            "CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=INNODB",
            "CREATE TABLE `recovery_parent` (`id` INT PRIMARY KEY) ENGINE=INNODB",
            "CREATE TABLE `recovery_child` (`id` INT PRIMARY KEY, `parent_id` INT, "
            "CONSTRAINT `recovery_child_parent_fk` FOREIGN KEY (`parent_id`) "
            "REFERENCES `recovery_parent` (`id`) ON DELETE CASCADE) ENGINE=INNODB",
            "CREATE TABLE `recovery_unique` (`value` VARCHAR(64), "
            "UNIQUE KEY `recovery_value_unique` (`value`)) ENGINE=INNODB",
            "INSERT INTO `fop_files` (`filename`, `data`, `uploaded_at`) VALUES "
            "('Pruefung-Mueller.pdf', FROM_BASE64('AP8BgD8='), 1788172496), "
            "('leer.txt', FROM_BASE64(''), 1788172497)",
            "INSERT INTO `dokument` (`file`, `kommentar`) VALUES "
            "(1, 'Zeile 1\\nZeile 2 mit Umlauten: äöüß'), (2, NULL)",
            "INSERT INTO `recovery_parent` VALUES (1)",
            "INSERT INTO `recovery_child` VALUES (1, 1)",
            "INSERT INTO `recovery_unique` VALUES ('only-once')",
            "CREATE TRIGGER `recovery_fop_files_ai` AFTER INSERT ON `fop_files` "
            "FOR EACH ROW INSERT INTO `recovery_audit` (`file_id`, `event_name`) "
            "VALUES (NEW.`ID`, 'insert')",
        ]
    )
    expected_columns.update(
        {
            "recovery_audit": ["file_id", "event_name"],
            "recovery_parent": ["id"],
            "recovery_child": ["id", "parent_id"],
            "recovery_unique": ["value"],
        }
    )
    return ";\n".join(statements) + ";\n", expected_columns


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
        f"--defaults-file={auth_file}",
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
        f"--defaults-file={auth_file}",
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
                 database: str = SOURCE_DATABASE,
                 password_option: str | None = None) -> list[str]:
    command = [
        dump_tool,
        f"--defaults-extra-file={auth_file}",
        "--lock-tables",
        "--complete-insert",
        "--add-drop-table",
        "--quick",
        "--quote-names",
        "--result-file",
        str(dump_path),
        database,
    ]
    if password_option:
        command.insert(2, password_option)
    return command


def mysql_client_args(client: str, auth_file: Path) -> list[str]:
    return [client, f"--defaults-file={auth_file}"]


def authenticated_client_args(client: str, auth_file: Path,
                              password_option: str | None) -> list[str]:
    command = mysql_client_args(client, auth_file)
    if password_option:
        command.append(password_option)
    return command


def verify_restore(client: str, auth_file: Path, database: str,
                   expected_columns: dict[str, list[str]] | None,
                   password_option: str | None = None) -> None:
    if expected_columns is not None:
        restored_tables = set(mysql_lines(
            client, auth_file, "SHOW TABLES", database, password_option
        ))
        if restored_tables != set(expected_columns):
            raise RecoveryError(
                "restored application table set differs\n"
                f"expected: {sorted(expected_columns)}\n"
                f"actual:   {sorted(restored_tables)}"
            )
        for table, columns in expected_columns.items():
            rows = mysql_lines(
                client,
                auth_file,
                f"SHOW COLUMNS FROM {quote_identifier(table)}",
                database,
                password_option,
            )
            restored_columns = [row.split("\t", 1)[0] for row in rows]
            if restored_columns != columns:
                raise RecoveryError(
                    f"restored columns for {table} = {restored_columns}, want {columns}"
                )

    actual = mysql_lines(
        client, auth_file, VERIFY_SQL, database, password_option
    )
    if actual != EXPECTED_VERIFY_LINES:
        raise RecoveryError(
            "restored data differs from the MemCP fixture\n"
            f"expected: {EXPECTED_VERIFY_LINES}\nactual:   {actual}"
        )

    post_restore = mysql_lines(
        client,
        auth_file,
        """
        INSERT INTO `fop_files` (`filename`, `data`, `uploaded_at`)
          VALUES ('nach-restore.txt', FROM_BASE64('AQI='), 1788172498);
        SELECT LAST_INSERT_ID();
        DELETE FROM `recovery_parent` WHERE `id` = 1;
        SELECT COUNT(*) FROM `recovery_child` WHERE `parent_id` = 1;
        SELECT COUNT(*), MIN(`file_id`), MIN(`event_name`) FROM `recovery_audit`;
        """,
        database,
        password_option,
    )
    if post_restore != ["3", "0", "1\t3\tinsert"]:
        raise RecoveryError(
            "AUTO_INCREMENT, foreign-key cascade, or restored trigger is incorrect: "
            f"{post_restore}"
        )

    duplicate = run(
        authenticated_client_args(client, auth_file, password_option) + [database],
        input_text=(
            "INSERT INTO `recovery_unique` (`value`) VALUES ('only-once')"
        ),
        check=False,
    )
    if duplicate.returncode == 0:
        raise RecoveryError("restored unique index accepted a duplicate value")
    duplicate_detail = duplicate.stderr or duplicate.stdout
    if "Duplicate entry" not in duplicate_detail or "1062" not in duplicate_detail:
        raise RecoveryError(
            "duplicate-value check failed for an unrelated reason:\n"
            + duplicate_detail.strip()
        )


def restore_and_verify(client: str, auth_file: Path, database: str,
                       dump_path: Path,
                       expected_columns: dict[str, list[str]],
                       password_option: str | None = None) -> None:
    command = authenticated_client_args(client, auth_file, password_option)
    run(
        command,
        input_text=f"CREATE DATABASE {quote_identifier(database)} CHARACTER SET utf8mb4",
    )
    run_with_stdin_file(command + [database], dump_path)
    verify_restore(
        client,
        auth_file,
        database,
        expected_columns,
        password_option,
    )


def main() -> int:
    args = parse_args()
    require_command(args.memcp_binary)
    require_command(args.mariadb_client)
    require_command(args.mysqldump)
    if not Path(args.app).is_file():
        raise RecoveryError(f"Scheme application does not exist: {args.app}")

    destination_password = os.environ.get("MEMCP_RECOVERY_MARIADB_PASSWORD", "")
    run_id = f"{os.getpid()}_{secrets.token_hex(4)}"
    target_databases = {
        "memcp_to_mariadb": f"recovery_mmaria_{run_id}",
        "memcp_to_memcp": f"recovery_mmemcp_{run_id}",
        "mariadb_to_mariadb": f"recovery_mariamaria_{run_id}",
        "mariadb_to_memcp": f"recovery_mariamemcp_{run_id}",
    }
    source_mysql_port = free_port()
    source_api_port = free_port()
    restore_mysql_port = free_port()
    restore_api_port = free_port()
    fixture_sql = FIXTURE_SQL
    expected_columns = FIXTURE_COLUMNS
    if args.application_schema_json is not None:
        fixture_sql, expected_columns = application_fixture(args.application_schema_json)

    with tempfile.TemporaryDirectory(prefix="memcp-mysqldump-recovery-") as temp_name:
        temp = Path(temp_name)
        dump_env = os.environ.copy()
        dump_env["HOME"] = str(temp)
        source_data_dir = temp / "source-data"
        source_data_dir.mkdir()
        restore_data_dir = temp / "restore-data"
        restore_data_dir.mkdir()
        source_auth = temp / "source.cnf"
        source_dump_auth = temp / "source-dump.cnf"
        destination_auth = temp / "destination.cnf"
        restore_auth = temp / "restore.cnf"
        memcp_dump_path = temp / "memcp-recovery.sql"
        memcp_system_dump_path = temp / "memcp-system.sql"
        memcp_statistics_dump_path = temp / "memcp-system-statistic.sql"
        mariadb_dump_path = temp / "mariadb-recovery.sql"
        source_log = temp / "memcp-source.log"
        restore_log = temp / "memcp-restore.log"
        client_auth_file(
            source_auth,
            host="127.0.0.1",
            port=source_mysql_port,
            socket_path=None,
            user="root",
            password="admin",
        )
        dump_username = f"recovery_dump_{run_id}"
        dump_password = secrets.token_urlsafe(18)
        client_auth_file(
            source_dump_auth,
            host="127.0.0.1",
            port=source_mysql_port,
            socket_path=None,
            user=dump_username,
            password=dump_password,
        )
        client_auth_file(
            destination_auth,
            host=args.mariadb_host,
            port=args.mariadb_port,
            socket_path=args.mariadb_socket,
            user=args.mariadb_user,
            password=destination_password,
        )
        client_auth_file(
            restore_auth,
            host="127.0.0.1",
            port=restore_mysql_port,
            socket_path=None,
            user="root",
            password="admin",
        )

        with source_log.open("w") as log:
            source_process = subprocess.Popen(
                [
                    args.memcp_binary,
                    "-data",
                    str(source_data_dir),
                    f"--api-port={source_api_port}",
                    f"--mysql-port={source_mysql_port}",
                    "--mysql-socket=",
                    "--no-repl",
                    args.app,
                ],
                stdin=subprocess.DEVNULL,
                stdout=log,
                stderr=subprocess.STDOUT,
                text=True,
            )
        restore_process: subprocess.Popen[str] | None = None
        try:
            wait_for_memcp(
                args.mariadb_client, source_auth, source_process, source_log
            )
            mysql_lines(
                args.mariadb_client,
                source_auth,
                fixture_sql,
                password_option="--password=admin",
            )
            mysql_lines(
                args.mariadb_client,
                source_auth,
                (
                    f"CREATE USER {quote_identifier(dump_username)} "
                    f"IDENTIFIED BY '{dump_password}'; "
                    "GRANT SELECT, LOCK TABLES, SHOW VIEW, TRIGGER ON "
                    f"{quote_identifier(SOURCE_DATABASE)}.* TO "
                    f"{quote_identifier(dump_username)}; "
                    "GRANT SELECT, LOCK TABLES, SHOW VIEW, TRIGGER ON "
                    f"system.* TO {quote_identifier(dump_username)}; "
                    "GRANT SELECT, LOCK TABLES, SHOW VIEW, TRIGGER ON "
                    f"system_statistic.* TO {quote_identifier(dump_username)}"
                ),
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

            dump_identity = mysql_lines(
                args.mariadb_client,
                source_dump_auth,
                "SELECT CURRENT_USER()",
                SOURCE_DATABASE,
            )
            if dump_identity != [f"{dump_username}@%"]:
                raise RecoveryError(
                    f"dump connection used unexpected identity: {dump_identity}"
                )

            try:
                run(dump_command(
                    args.mysqldump,
                    source_dump_auth,
                    memcp_dump_path,
                ), env=dump_env)
                run(dump_command(
                    args.mysqldump,
                    source_dump_auth,
                    memcp_system_dump_path,
                    "system",
                ), env=dump_env)
                run(dump_command(
                    args.mysqldump,
                    source_dump_auth,
                    memcp_statistics_dump_path,
                    "system_statistic",
                ), env=dump_env)
            except RecoveryError as error:
                log_tail = source_log.read_text(errors="replace")[-4000:]
                raise RecoveryError(
                    f"{error}\nMemCP log tail:\n{log_tail}"
                ) from error

            mariadb_client = mysql_client_args(args.mariadb_client, destination_auth)
            run(
                mariadb_client,
                input_text=f"DROP DATABASE IF EXISTS {quote_identifier(SOURCE_DATABASE)}",
            )
            run(mariadb_client, input_text=fixture_sql)
            mariadb_source_values = mysql_lines(
                args.mariadb_client,
                destination_auth,
                VERIFY_SQL,
                SOURCE_DATABASE,
            )
            if mariadb_source_values != EXPECTED_VERIFY_LINES:
                raise RecoveryError(
                    f"MariaDB fixture verification failed: {mariadb_source_values}"
                )

            run(dump_command(
                args.mysqldump,
                destination_auth,
                mariadb_dump_path,
            ), env=dump_env)
            print(
                "plain dumps from MemCP application/system databases and MariaDB "
                "completed successfully"
            )

            memcp_to_mariadb = target_databases["memcp_to_mariadb"]
            restore_and_verify(
                args.mariadb_client,
                destination_auth,
                memcp_to_mariadb,
                memcp_dump_path,
                expected_columns,
            )
            print(
                "PASS: MemCP dump -> MariaDB restore with complete integrity checks"
            )

            with restore_log.open("w") as log:
                restore_process = subprocess.Popen(
                    [
                        args.memcp_binary,
                        "-data",
                        str(restore_data_dir),
                        f"--api-port={restore_api_port}",
                        f"--mysql-port={restore_mysql_port}",
                        "--mysql-socket=",
                        "--no-repl",
                        args.app,
                    ],
                    stdin=subprocess.DEVNULL,
                    stdout=log,
                    stderr=subprocess.STDOUT,
                    text=True,
                )
            wait_for_memcp(
                args.mariadb_client, restore_auth, restore_process, restore_log
            )
            try:
                restore_and_verify(
                    args.mariadb_client,
                    restore_auth,
                    target_databases["memcp_to_memcp"],
                    memcp_dump_path,
                    expected_columns,
                    "--password=admin",
                )
                print(
                    "PASS: MemCP dump -> MemCP restore with complete integrity checks"
                )

                mariadb_to_mariadb = target_databases["mariadb_to_mariadb"]
                restore_and_verify(
                    args.mariadb_client,
                    destination_auth,
                    mariadb_to_mariadb,
                    mariadb_dump_path,
                    expected_columns,
                )
                print(
                    "PASS: MariaDB dump -> MariaDB restore with complete integrity checks"
                )

                restore_and_verify(
                    args.mariadb_client,
                    restore_auth,
                    target_databases["mariadb_to_memcp"],
                    mariadb_dump_path,
                    expected_columns,
                    "--password=admin",
                )
                print(
                    "PASS: MariaDB dump -> MemCP restore with complete integrity checks"
                )
            except RecoveryError as error:
                log_tail = restore_log.read_text(errors="replace")[-4000:]
                raise RecoveryError(
                    f"{error}\nMemCP restore log tail:\n{log_tail}"
                ) from error
            return 0
        finally:
            for database in [
                SOURCE_DATABASE,
                target_databases["memcp_to_mariadb"],
                target_databases["mariadb_to_mariadb"],
            ]:
                run(
                    mysql_client_args(args.mariadb_client, destination_auth),
                    input_text=f"DROP DATABASE IF EXISTS {quote_identifier(database)}",
                    check=False,
                )
            if restore_process is not None and restore_process.poll() is None:
                restore_process.send_signal(signal.SIGTERM)
                try:
                    restore_process.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    restore_process.kill()
                    restore_process.wait(timeout=5)
            if source_process.poll() is None:
                source_process.send_signal(signal.SIGTERM)
                try:
                    source_process.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    source_process.kill()
                    source_process.wait(timeout=5)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except RecoveryError as error:
        print(f"FAIL: {error}", file=sys.stderr)
        raise SystemExit(1)
