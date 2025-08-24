#!/usr/bin/env python3
"""
MemCP API Test Suite

Tests the MemCP database via HTTP REST API.
This script assumes memcp is running on localhost:4321 with MySQL protocol on port 3307.
"""

import requests
import json
import time
import subprocess
import os
import sys
from base64 import b64encode

class MemCPTester:
    def __init__(self, base_url="http://localhost:4321", username="root", password="admin"):
        self.base_url = base_url
        self.username = username
        self.password = password
        self.auth_header = self._create_auth_header()
        self.test_count = 0
        self.test_passed = 0
        
    def _create_auth_header(self):
        """Create HTTP Basic Auth header"""
        credentials = f"{self.username}:{self.password}"
        encoded = b64encode(credentials.encode()).decode()
        return {"Authorization": f"Basic {encoded}"}
    
    def assert_test(self, condition, message):
        """Simple assertion helper"""
        self.test_count += 1
        if condition:
            self.test_passed += 1
            print(f"✓ Test {self.test_count}: {message}")
        else:
            print(f"✗ Test {self.test_count}: {message}")
    
    def execute_sql(self, database, query):
        """Execute SQL query via HTTP API"""
        try:
            url = f"{self.base_url}/sql/{database}"
            response = requests.post(url, data=query, headers=self.auth_header, timeout=10)
            return response
        except Exception as e:
            print(f"Error executing SQL: {e}")
            return None
    
    def test_basic_connectivity(self):
        """Test basic API connectivity"""
        print("Testing basic connectivity...")
        try:
            response = requests.get(self.base_url, timeout=5)
            self.assert_test(response.status_code in [200, 301, 404], "API endpoint reachable")
        except Exception as e:
            self.assert_test(False, f"API endpoint reachable (Error: {e})")
    
    def test_sql_parsing(self):
        """Test SQL parsing and basic queries"""
        print("\nTesting SQL parsing and execution...")
        
        # Test simple SELECT
        response = self.execute_sql("system", "SELECT 1 as test_value")
        self.assert_test(response and response.status_code == 200, "Simple SELECT query")
        
        # Test SELECT from system table
        response = self.execute_sql("system", "SELECT COUNT(*) FROM user")
        self.assert_test(response and response.status_code == 200, "SELECT COUNT from system.user")
        
        # Test CREATE DATABASE
        response = self.execute_sql("system", "CREATE DATABASE test_db")
        self.assert_test(response and response.status_code == 200, "CREATE DATABASE")
        
        # Test CREATE TABLE
        response = self.execute_sql("test_db", "CREATE TABLE test_users (id INT PRIMARY KEY, name VARCHAR(100), age INT)")
        self.assert_test(response and response.status_code == 200, "CREATE TABLE")
        
        # Test INSERT
        response = self.execute_sql("test_db", "INSERT INTO test_users (id, name, age) VALUES (1, 'Alice', 25)")
        self.assert_test(response and response.status_code == 200, "INSERT single row")
        
        response = self.execute_sql("test_db", "INSERT INTO test_users (id, name, age) VALUES (2, 'Bob', 30), (3, 'Charlie', 35)")
        self.assert_test(response and response.status_code == 200, "INSERT multiple rows")
        
        # Test SELECT with data
        response = self.execute_sql("test_db", "SELECT * FROM test_users ORDER BY id")
        self.assert_test(response and response.status_code == 200, "SELECT with ORDER BY")
        
        # Test aggregate functions
        response = self.execute_sql("test_db", "SELECT COUNT(*) FROM test_users")
        self.assert_test(response and response.status_code == 200, "SELECT COUNT(*)")
        
        response = self.execute_sql("test_db", "SELECT MAX(age) FROM test_users")
        self.assert_test(response and response.status_code == 200, "SELECT MAX()")
        
        # Test WHERE clause
        response = self.execute_sql("test_db", "SELECT name FROM test_users WHERE age > 25")
        self.assert_test(response and response.status_code == 200, "SELECT with WHERE")
        
        # Test UPDATE
        response = self.execute_sql("test_db", "UPDATE test_users SET age = 26 WHERE name = 'Alice'")
        self.assert_test(response and response.status_code == 200, "UPDATE query")
        
        # Test DELETE
        response = self.execute_sql("test_db", "DELETE FROM test_users WHERE age > 35")
        self.assert_test(response and response.status_code == 200, "DELETE query")
        
        # Test LIKE operator
        response = self.execute_sql("test_db", "SELECT name FROM test_users WHERE name LIKE 'A%'")
        self.assert_test(response and response.status_code == 200, "SELECT with LIKE")
        
        # Test LIMIT
        response = self.execute_sql("test_db", "SELECT * FROM test_users LIMIT 2")
        self.assert_test(response and response.status_code == 200, "SELECT with LIMIT")
        
        # Cleanup
        response = self.execute_sql("system", "DROP DATABASE test_db")
        self.assert_test(response and response.status_code == 200, "DROP DATABASE cleanup")
    
    def test_error_handling(self):
        """Test error handling for invalid SQL"""
        print("\nTesting error handling...")
        
        # Test invalid syntax
        response = self.execute_sql("system", "INVALID SQL SYNTAX")
        print(f"  Invalid SQL - Status: {response.status_code if response else 'None'}, Body: {response.text[:100] if response else 'None'}")
        self.assert_test(response and response.status_code != 200, "Invalid SQL syntax rejected")
        
        # Test non-existent table
        response = self.execute_sql("system", "SELECT * FROM non_existent_table")
        print(f"  Non-existent table - Status: {response.status_code if response else 'None'}, Body: {response.text[:100] if response else 'None'}")
        self.assert_test(response and response.status_code != 200, "Non-existent table rejected")
        
        # Test non-existent database
        response = self.execute_sql("non_existent_db", "SELECT 1")
        print(f"  Non-existent DB - Status: {response.status_code if response else 'None'}, Body: {response.text[:100] if response else 'None'}")
        self.assert_test(response and response.status_code != 200, "Non-existent database rejected")
    
    def run_all_tests(self):
        """Run all tests"""
        print("=" * 50)
        print("MemCP API Test Suite")
        print("=" * 50)
        
        self.test_basic_connectivity()
        self.test_sql_parsing()
        self.test_error_handling()
        
        print("\n" + "=" * 50)
        print(f"Test Results: {self.test_passed}/{self.test_count} passed")
        print("=" * 50)
        
        return self.test_passed == self.test_count

def wait_for_memcp(timeout=30):
    """Wait for MemCP to be ready"""
    print("Waiting for MemCP to start...")
    for i in range(timeout):
        try:
            response = requests.get("http://localhost:4321", timeout=2)
            print("MemCP is ready!")
            return True
        except:
            time.sleep(1)
            if i % 5 == 0:
                print(f"Still waiting... ({i}s)")
    return False

if __name__ == "__main__":
    # Check if MemCP is running, if not start it
    try:
        requests.get("http://localhost:4321", timeout=2)
        print("MemCP is already running")
    except:
        print("Starting MemCP...")
        # Start MemCP in background with stdin pipe to keep it running
        process = subprocess.Popen(
            ["./memcp", "-data", "/tmp/memcp-test-data"],
            cwd=os.path.dirname(os.path.abspath(__file__)),
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE
        )
        
        if not wait_for_memcp():
            print("Failed to start MemCP")
            sys.exit(1)
    
    # Run tests
    tester = MemCPTester()
    success = tester.run_all_tests()
    
    sys.exit(0 if success else 1)