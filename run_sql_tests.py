#!/usr/bin/env python3
"""
MemCP SQL Test Runner

Executes structured SQL tests from YAML specification files.
This provides a declarative, maintainable way to define comprehensive SQL tests.
"""

import yaml
import json
import requests
import subprocess
import time
import sys
import os
from pathlib import Path
from base64 import b64encode
from typing import Dict, List, Any, Optional
from urllib.parse import quote
import re

# Global flag for connect-only mode
is_connect_only_mode = False

class SQLTestRunner:
    def __init__(self, base_url="http://localhost:4321", username="root", password="admin"):
        self.base_url = base_url
        self.username = username
        self.password = password
        self.auth_header = self._create_auth_header()
        self.test_count = 0
        self.test_passed = 0
        self.failed_tests = []
        self.setup_operations = []  # Track setup operations for recovery
        self.current_database = None  # Track current database for recovery
        
    def _create_auth_header(self):
        """Create HTTP Basic Auth header"""
        credentials = f"{self.username}:{self.password}"
        encoded = b64encode(credentials.encode()).decode()
        return {"Authorization": f"Basic {encoded}"}
    
    def execute_sql(self, database: str, query: str) -> Optional[requests.Response]:
        """Execute SQL query via HTTP API with database recovery"""
        try:
            # URL-encode database name to handle hyphens and special characters
            encoded_db = quote(database, safe='')
            url = f"{self.base_url}/sql/{encoded_db}"
            response = requests.post(url, data=query, headers=self.auth_header, timeout=10)
            
            # Check if database was lost due to MemCP bug - attempt recovery
            if response and 'database ' + database + ' does not exist' in response.text:
                print(f"    🔧 Database {database} was lost! Attempting full recovery...")
                # Try to recreate the database
                create_db_sql = f"CREATE DATABASE IF NOT EXISTS {database}"
                recovery_response = requests.post(f"{self.base_url}/sql/system", 
                                                data=create_db_sql, 
                                                headers=self.auth_header, 
                                                timeout=10)
                if recovery_response and recovery_response.status_code == 200:
                    print(f"    ✅ Database {database} recovered")
                    
                    # Also recreate setup operations (tables, etc.)
                    if self.setup_operations and self.current_database == database:
                        print(f"    🔧 Recreating {len(self.setup_operations)} setup operations...")
                        for step in self.setup_operations:
                            setup_response = requests.post(url, data=step['sql'], 
                                                         headers=self.auth_header, timeout=10)
                            if setup_response and setup_response.status_code == 200:
                                print(f"    ✅ Recreated: {step['action']}")
                            else:
                                print(f"    ⚠️  Failed to recreate: {step['action']}")
                    
                    print(f"    🔄 Retrying original query...")
                    # Retry the original query
                    response = requests.post(url, data=query, headers=self.auth_header, timeout=10)
                else:
                    print(f"    ❌ Failed to recover database {database}")
            
            return response
        except Exception as e:
            print(f"Error executing SQL: {e}")
            return None
    
    def execute_sparql(self, database: str, query: str) -> Optional[requests.Response]:
        """Execute SPARQL query via HTTP API"""
        try:
            # URL-encode database name to handle hyphens and special characters
            encoded_db = quote(database, safe='')
            url = f"{self.base_url}/rdf/{encoded_db}"
            response = requests.post(url, data=query, headers=self.auth_header, timeout=10)
            return response
        except Exception as e:
            print(f"Error executing SPARQL: {e}")
            return None
    
    def execute_scheme(self, scheme_code: str) -> Optional[requests.Response]:
        """Execute Scheme code via HTTP API"""
        try:
            url = f"{self.base_url}/scheme"
            response = requests.post(url, data=scheme_code, headers=self.auth_header, timeout=10)
            return response
        except Exception as e:
            print(f"Error executing Scheme: {e}")
            return None
    
    def load_ttl(self, database: str, ttl_data: str) -> bool:
        """Load TTL data into RDF table"""
        try:
            # First ensure database exists
            create_db_sql = f"CREATE DATABASE IF NOT EXISTS {database}"
            response = self.execute_sql("system", create_db_sql)
            if not response or response.status_code != 200:
                print(f"Failed to create database: {response.text if response else 'No response'}")
                return False
            
            # Load TTL data using the new /rdf/{database}/load_ttl endpoint
            encoded_db = quote(database, safe='')
            url = f"{self.base_url}/rdf/{encoded_db}/load_ttl"
            response = requests.post(url, data=ttl_data, headers=self.auth_header, timeout=10)
            
            if response and response.status_code == 200:
                return True
            else:
                print(f"Failed to load TTL data: {response.text if response else 'No response'}")
                return False
            
        except Exception as e:
            print(f"Error loading TTL data: {e}")
            return False
    
    def parse_jsonl_response(self, response: requests.Response) -> Optional[List[Dict]]:
        """Parse JSONL response into list of dictionaries"""
        if not response:
            return None
        
        try:
            response_text = response.text.strip()
            
            # Handle empty response (valid for INSERT/UPDATE/DELETE)
            if not response_text:
                return []
                
            lines = response_text.split('\n')
            results = []
            for line in lines:
                line = line.strip()
                if line:
                    try:
                        results.append(json.loads(line))
                    except json.JSONDecodeError as json_err:
                        # Non-JSON lines are likely error messages - don't skip them!
                        # The validation logic will handle them properly
                        # Only show verbose errors if this is an unexpected error
                        continue
            return results
        except Exception as e:
            print(f"Error parsing JSONL response: {e}")
            print(f"Response text: '{response.text}'")
            return None

    def run_setup(self, setup_steps: List[Dict], database: str) -> bool:
        """Execute setup steps"""
        print("🔧 Running setup...")
        self.setup_operations = []  # Reset setup operations for new test file
        self.current_database = database
        
        for step in setup_steps:
            print(f"  - {step['action']}")
            # Store setup operation for potential recovery
            self.setup_operations.append(step)
            
            response = self.execute_sql(database, step['sql'])
            if not response or response.status_code not in [200, 500]:  # 500 might be expected for CREATE DATABASE IF NOT EXISTS
                print(f"    ❌ Setup step failed: {step['action']}")
                print(f"        Response: {response.text if response else 'No response'}")
                return False
        print("  ✅ Setup completed")
        return True
    
    def run_cleanup(self, cleanup_steps: List[Dict], database: str) -> bool:
        """Execute cleanup steps"""
        print("\n🧹 Running cleanup...")
        for step in cleanup_steps:
            print(f"  - {step['action']}")
            response = self.execute_sql(database, step['sql'])
            # Don't fail on cleanup errors
            if response and response.status_code == 200:
                print(f"    ✅ Cleanup step succeeded")
            else:
                print(f"    ⚠️  Cleanup step had issues (continuing anyway)")
        return True

    def validate_expectation(self, test_case: Dict, response: requests.Response, results: Optional[List[Dict]]) -> bool:
        """Validate test results against expectations"""
        expect = test_case.get('expect', {})
        
        # Check for expected errors
        if expect.get('error'):
            if response.status_code != 200 or 'Error' in response.text:
                # Expected error - pass silently
                return True
            else:
                print(f"    ❌ Expected error but got success (HTTP {response.status_code})")
                print(f"        Response: {response.text[:200]}")
                return False
        
        # Check for unexpected errors (when we expected success)
        if 'Error' in response.text:
            print(f"    ❌ Unexpected error in response")
            print(f"        Error: {response.text[:200]}")
            return False
            
        # Check HTTP status
        if response.status_code != 200:
            print(f"    ❌ HTTP {response.status_code}: {response.text[:100]}")
            return False
        
        # Check affected rows (for INSERT/UPDATE/DELETE/DDL)
        if 'affected_rows' in expect:
            expected_affected = expect['affected_rows']
            if results and len(results) > 0 and 'affected_rows' in results[0]:
                actual_affected = results[0]['affected_rows']
                if actual_affected != expected_affected:
                    print(f"    ❌ Expected {expected_affected} affected rows, got {actual_affected}")
                    return False
            else:
                # For DDL operations that don't return data, just check they succeeded
                return True
            
        # Check result data
        if results is None:
            print(f"    ❌ Failed to parse response")
            return False
            
        # Check row count
        expected_rows = expect.get('rows')
        if expected_rows is not None:
            if len(results) != expected_rows:
                print(f"    ❌ Expected {expected_rows} rows, got {len(results)}")
                return False
                
        # Check specific data values
        expected_data = expect.get('data')
        if expected_data:
            for i, expected_row in enumerate(expected_data):
                if i >= len(results):
                    print(f"    ❌ Missing expected row {i}")
                    return False
                    
                actual_row = results[i]
                for key, expected_value in expected_row.items():
                    actual_value = actual_row.get(key)
                    
                    # Handle floating point comparison
                    if isinstance(expected_value, float) and isinstance(actual_value, (int, float)):
                        if abs(actual_value - expected_value) > 0.01:
                            print(f"    ❌ Row {i}: Expected {key}={expected_value}, got {actual_value}")
                            return False
                    elif actual_value != expected_value:
                        print(f"    ❌ Row {i}: Expected {key}={expected_value}, got {actual_value}")
                        return False
        
        return True

    def run_test_case(self, test_case: Dict, database: str) -> bool:
        """Execute a single test case (SQL or SPARQL)"""
        self.test_count += 1
        name = test_case.get('name', f'Test {self.test_count}')
        
        # Check if this is a SPARQL test case
        if 'sparql' in test_case:
            return self.run_sparql_test_case(test_case, database)
        
        # Regular SQL test case
        sql = test_case['sql'].strip()
        
        print(f"\n📋 Test {self.test_count}: {name}")
        print(f"    SQL: {sql[:80]}{'...' if len(sql) > 80 else ''}")
        
        # Execute SQL
        response = self.execute_sql(database, sql)
        if not response:
            print(f"    ❌ Failed to execute SQL")
            self.failed_tests.append(name)
            return False
            
        # Parse results
        results = self.parse_jsonl_response(response)
        
        # Validate expectations
        if self.validate_expectation(test_case, response, results):
            print(f"    ✅ Passed")
            self.test_passed += 1
            return True
        else:
            print(f"    ❌ Failed")
            self.failed_tests.append(name)
            return False
    
    def run_sparql_test_case(self, test_case: Dict, database: str) -> bool:
        """Execute a SPARQL test case"""
        name = test_case.get('name', f'SPARQL Test {self.test_count}')
        sparql = test_case['sparql'].strip()
        
        print(f"\n📋 Test {self.test_count}: {name}")
        print(f"    SPARQL: {sparql[:80]}{'...' if len(sparql) > 80 else ''}")
        
        # Load TTL data if provided
        if 'ttl_data' in test_case:
            ttl_data = test_case['ttl_data'].strip()
            print(f"    Loading TTL data ({len(ttl_data)} chars)")
            if not self.load_ttl(database, ttl_data):
                print(f"    ❌ Failed to load TTL data")
                self.failed_tests.append(name)
                return False
        
        # Execute SPARQL
        response = self.execute_sparql(database, sparql)
        if not response:
            print(f"    ❌ Failed to execute SPARQL")
            self.failed_tests.append(name)
            return False
            
        # Parse results
        results = self.parse_jsonl_response(response)
        
        # Validate expectations
        if self.validate_expectation(test_case, response, results):
            print(f"    ✅ Passed")
            self.test_passed += 1
            return True
        else:
            print(f"    ❌ Failed")
            self.failed_tests.append(name)
            return False

    def cleanup_test_database(self, database: str) -> None:
        """Clean up test database between test runs"""
        try:
            # In connect-only mode, we need gentler cleanup
            # since multiple test files share the same MemCP instance
            global is_connect_only_mode
            if is_connect_only_mode:
                # First, clean up any test-created databases that might interfere
                try:
                    # Get list of all databases
                    response = self.execute_sql("system", "SHOW DATABASES")
                    if response and response.status_code == 200:
                        try:
                            databases_data = response.json()
                            if isinstance(databases_data, dict) and 'data' in databases_data:
                                for row in databases_data['data']:
                                    if isinstance(row, dict):
                                        db_name = list(row.values())[0]  # Get first value from row
                                        # Drop any test databases that might cause interference (except system and our main test db)
                                        if db_name not in ['system', 'memcp-tests'] and db_name != database and ('test' in db_name.lower() or 'edge_cases' in db_name.lower()):
                                            drop_db_sql = f"DROP DATABASE IF EXISTS {db_name}"
                                            self.execute_sql("system", drop_db_sql)
                                            print(f"🧹 Removed interfering test database: {db_name}")
                        except Exception as e:
                            print(f"⚠️  Could not cleanup interfering databases: {e}")
                except:
                    pass
                
                # Now clean tables within the main test database
                try:
                    # Try to get list of tables and drop them individually
                    response = self.execute_sql(database, "SHOW TABLES")
                    if response and response.status_code == 200:
                        try:
                            tables_data = response.json()
                            if isinstance(tables_data, dict) and 'data' in tables_data:
                                for row in tables_data['data']:
                                    if isinstance(row, dict):
                                        table_name = list(row.values())[0]  # Get first value from row
                                        drop_table_sql = f"DROP TABLE IF EXISTS {table_name}"
                                        self.execute_sql(database, drop_table_sql)
                                print(f"🧹 Cleaned tables in database: {database}")
                            else:
                                print(f"🧹 No tables to clean in database: {database}")
                        except:
                            print(f"🧹 Performed basic cleanup for database: {database}")
                except:
                    # Database might not exist yet, which is fine - it will be created after cleanup
                    print(f"🧹 Database will be created fresh: {database}")
                return
            
            # Traditional mode: drop test database to ensure clean state  
            drop_db_sql = f"DROP DATABASE IF EXISTS {database}"
            response = self.execute_sql("system", drop_db_sql)
            if response and response.status_code == 200:
                print(f"🧹 Cleaned up database: {database}")
        except Exception as e:
            print(f"⚠️  Warning: Failed to cleanup database {database}: {e}")

    def run_test_spec(self, spec_file: str) -> bool:
        """Run tests from a YAML specification file"""
        print(f"📖 Loading test specification: {spec_file}")
        
        try:
            with open(spec_file, 'r') as f:
                spec = yaml.safe_load(f)
        except Exception as e:
            print(f"❌ Failed to load spec file: {e}")
            return False
            
        metadata = spec.get('metadata', {})
        database = metadata.get('database', 'memcp-tests')
        
        print(f"🎯 Running test suite: {metadata.get('description', 'SQL Tests')}")
        print(f"📊 Version: {metadata.get('version', 'unknown')}")
        print(f"💾 Database: {database}")
        
        # Clean up database before starting (ensure isolation)
        self.cleanup_test_database(database)
        
        # Ensure the test database exists (cleanup may have cleaned it)
        # Try without backticks first - MemCP might not handle them correctly
        create_db_sql = f"CREATE DATABASE IF NOT EXISTS {database}"
        response = self.execute_sql("system", create_db_sql)
        if not response or response.status_code != 200:
            print(f"❌ Failed to create test database: {database}")
            print(f"    Response: {response.text if response else 'No response'}")
            print(f"    SQL: {create_db_sql}")
            return False
        print(f"✅ Database ready: {database}")
        
        # Run setup
        setup_steps = spec.get('setup', [])
        if setup_steps and not self.run_setup(setup_steps, database):
            return False
            
        # Run test cases
        test_cases = spec.get('test_cases', [])
        print(f"\n🚀 Running {len(test_cases)} test cases...")
        
        for test_case in test_cases:
            self.run_test_case(test_case, database)
            
        # Run cleanup
        cleanup_steps = spec.get('cleanup', [])
        if cleanup_steps:
            self.run_cleanup(cleanup_steps, database)
            
        # Clean up database after tests (leave system clean)
        self.cleanup_test_database(database)
        
        # Print summary
        print(f"\n" + "="*60)
        print(f"📊 Test Results: {self.test_passed}/{self.test_count} passed")
        
        if self.failed_tests:
            print(f"❌ Failed tests:")
            for failed_test in self.failed_tests:
                print(f"   - {failed_test}")
        else:
            print("🎉 All tests passed!")
            
        print("="*60)
        
        return self.test_passed == self.test_count

def wait_for_memcp(port=4321, timeout=30) -> bool:
    """Wait for MemCP to be ready"""
    print(f"⏳ Waiting for MemCP to start on port {port}...")
    for i in range(timeout):
        try:
            response = requests.get(f"http://localhost:{port}", timeout=2)
            print("✅ MemCP is ready!")
            return True
        except:
            time.sleep(1)
            if i % 5 == 0 and i > 0:
                print(f"   Still waiting... ({i}s)")
    return False

def main():
    """Main entry point"""
    if len(sys.argv) < 2:
        print("Usage: python3 run_sql_tests.py <test_spec.yaml> [port] [--connect-only]")
        sys.exit(1)
        
    spec_file = sys.argv[1]
    if not os.path.exists(spec_file):
        print(f"❌ Test specification file not found: {spec_file}")
        sys.exit(1)
    
    # Parse optional port parameter
    port = 4321
    connect_only = False
    
    for i, arg in enumerate(sys.argv[2:], 2):
        if arg == "--connect-only":
            connect_only = True
        elif arg.isdigit():
            port = int(arg)
    
    base_url = f"http://localhost:{port}"
    memcp_process = None
    
    # Store connect_only flag globally for cleanup decisions
    global is_connect_only_mode
    is_connect_only_mode = connect_only
    
    if connect_only:
        # Connect-only mode: just try to connect, don't start MemCP
        print(f"🔗 Connecting to existing MemCP on port {port}...")
        try:
            requests.get(base_url, timeout=2)
            print(f"✅ Connected to MemCP on port {port}")
        except:
            print(f"❌ Failed to connect to MemCP on port {port}")
            print(f"   Make sure MemCP is running first")
            sys.exit(1)
    else:
        # Traditional mode: start MemCP if needed
        try:
            requests.get(base_url, timeout=2)
            print(f"✅ MemCP is already running on port {port}")
        except:
            print(f"🚀 Starting MemCP on port {port}...")
            
            memcp_process = subprocess.Popen([
                "./memcp", 
                "-data", f"/tmp/memcp-sql-tests-{port}",
                f"--api-port={port}",
                f"--mysql-port={port + 1000}",
                "--disable-mysql",  # API-only for testing
                "lib/main.scm"     # Specify main script
            ], 
                cwd=os.path.dirname(os.path.abspath(__file__)),
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            
            if not wait_for_memcp(port):
                print("❌ Failed to start MemCP")
                if memcp_process.poll() is not None:
                    # Process has exited, show error output
                    stderr = memcp_process.stderr.read()
                    if stderr:
                        print(f"Error: {stderr}")
                memcp_process.terminate()
                sys.exit(1)
    
    # Run tests
    runner = SQLTestRunner(base_url)
    success = runner.run_test_spec(spec_file)
    
    # Cleanup: Stop MemCP process only if we started it
    if not connect_only and memcp_process:
        print("\n🛑 Stopping MemCP...")
        try:
            # Send EOF to stdin to trigger graceful shutdown
            memcp_process.stdin.close()
            memcp_process.wait(timeout=10)
            print("✅ MemCP stopped gracefully")
        except subprocess.TimeoutExpired:
            print("⚠️  MemCP didn't stop gracefully, terminating...")
            memcp_process.terminate()
            memcp_process.wait(timeout=5)
        except:
            memcp_process.kill()
    
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()