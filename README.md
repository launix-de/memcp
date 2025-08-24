<h1 align="center">memcp: Made for Big Data</h1>
<h4 align="center">A High-Performance, Open-Source Columnar In-Memory Database</h4>
<h4 align="center">Protocol compatible drop-in replacement for MySQL</h4>

<div align="center">

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![MySQL](https://img.shields.io/badge/mysql-%2300f.svg?style=for-the-badge&logo=mysql&logoColor=white) 
![HTML5](https://img.shields.io/badge/html5-%23E34F26.svg?style=for-the-badge&logo=html5&logoColor=white)
![JavaScript](https://img.shields.io/badge/javascript-%23323330.svg?style=for-the-badge&logo=javascript&logoColor=%23F7DF1E)
![REST](https://img.shields.io/badge/REST-lime?style=for-the-badge) 
![JSON](https://img.shields.io/badge/JSON-8A2BE2?style=for-the-badge) 

<br>

![memcp >](assets/memcp-logo.svg?raw=true)

<h1 align="center">Introduction</h1>

<b>In modern server and mainframe hardware, the memory bandwidth between CPU and RAM has become the new bottleneck. In-RAM compression will be a mayor contribution towards solving that issue.</b>

### What is memcp?
memcp is an open-source, high-performance, columnar in-memory database that can handle both OLAP and OLTP workloads. It provides an alternative to proprietary analytical databases and aims to bring the benefits of columnar storage to the open-source world.

###
memcp is written in Golang and is designed to be portable and extensible, allowing developers to embed the database into their applications with ease. It is also designed with a focus on scalability and performance, making it a suitable choice for distributed applications.

</div>

<br>

### Features
- <b>fast:</b> MemCP is built with parallelization in mind. The parallelization pattern is made for minimal overhead.
- <b>efficient:</b> The average compression ratio is 1:5 (80% memory saving) compared to MySQL/MariaDB
- <b>modern:</b> MemCP is built for modern hardware with caches, NUMA memory, multicore CPUs, NVMe SSDs
- <b>versatile:</b> Use it in big mainframes to gain analytical performance, use it in embedded systems to conserve flash lifetime
- Columnar storage: Stores data column-wise instead of row-wise, which allows for better compression, faster query execution, and more efficient use of memory.
- In-memory database: Stores all data in memory, which allows for extremely fast query execution.
- Build fast REST APIs and microservices directly in the database (they are faster because there is no network connection / SQL layer in between)
- OLAP and OLTP support: Can handle both online analytical processing (OLAP) and online transaction processing (OLTP) workloads.
- Compression: Lots of compression formats are supported like bit-packing and dictionary encoding
- Scalability: Designed to scale on a single node with huge NUMA memory
- Adjustable persistency: Decide whether you want to persist a table or not or to just keep snapshots of a period of time

<hr>

Project Website: [memcp.org](https://www.memcp.org)

<h1 align="center"> Screenshots </h1>


<div align="center">

<img src="assets/shot1.png" alt="Benchmark" border="0">

<br>

<img src="assets/shot2.png" alt="mysql client connecting to memcp" border="0">

</div>

<hr>

<h1 align="center">Table of Contents</h1>

- [🚶 Getting Started](#getting-started-)
- [🧪 Testing](#testing-)
- [🚀 Deployment](#deployment-)  
- [🔒 Security](#securing-the-database-from-external-access-)
- [🌿 Contributing](#contributing-)
- [❓ How it Works](#how-it-works-)
- [📚 Further Reading](#further-reading-)

<hr>

<h1 align="center">Getting Started 🚶</h1>

<h2>Using Docker</h2>

```
# first time: build the image
docker build . -t memcp

# run with the interactive scheme shell for debugging and development
mkdir data
docker run -v data:/data -it -p 4321:4321 -p 3307:3307 memcp

# run a specific application
docker run -e PARAMS="lib/main.scm apps/minigame.scm" -v data:/data -it -p 4321:4321 -p 3307:3307 memcp

# run with custom ports
docker run -v data:/data -it -p 8080:8080 -p 3308:3308 memcp --api-port=8080 --mysql-port=3308

# run API-only (no MySQL protocol)
docker run -v data:/data -it -p 8080:8080 memcp --api-port=8080 --disable-mysql

# run for productive use
mkdir /var/memcp
docker run -v /var/memcp:/data -di -p 4321:4321 -p 3307:3307 --restart unless-stopped memcp


```

<h2>Compile From Source</h2>

Make sure, `go` is installed on your computer.

Compile the project with

```bash
go get # installs dependencies
make   # executes go build
# or alternatively:
go build -o memcp
```

Run the engine with default settings:

```bash
./memcp
```

<h3>Command Line Arguments</h3>

MemCP supports various command-line arguments for flexible deployment:

```bash
# Basic usage
./memcp [options] [script_files...]

# Built-in Go flags
./memcp -data /path/to/data           # Set data directory (default: ./data)
./memcp -profile profile.out          # Enable CPU profiling
./memcp -wd /working/directory        # Set working directory for imports
./memcp -c "(print 'Hello World')"   # Execute Scheme command

# MemCP-specific arguments (handled by Scheme)
./memcp --api-port=8080              # HTTP API port (default: 4321)
./memcp --mysql-port=3308            # MySQL protocol port (default: 3307)
./memcp --disable-mysql              # Disable MySQL protocol server
./memcp --disable-api                # Disable HTTP API server

# Examples
./memcp --api-port=8080 --disable-mysql        # API-only on port 8080
./memcp --mysql-port=3308 --disable-api        # MySQL-only on port 3308
./memcp -data /var/memcp --api-port=4322       # Custom data dir and API port
```

<h3>Environment Variables</h3>

You can also use environment variables (command-line args take precedence):

```bash
export PORT=8080           # HTTP API port
export MYSQL_PORT=3308     # MySQL protocol port  
export DISABLE_MYSQL=true  # Disable MySQL server
./memcp
```

<h2>MemCP Scheme Shell</h2>

It will drop you at the scheme shell:

<pre><span style="color: #12488B"><b>~/memcp</b></span>$ ./memcp
memcp Copyright (C) 2023   Carl-Philip Hänsch
    This program comes with ABSOLUTELY NO WARRANTY;
    This is free software, and you are welcome to redistribute it
    under certain conditions;
Welcome to memcp
Hello World
MySQL server listening on port 3307 (connect with mysql -P 3307 -u user -p)
listening on http://localhost:4321
<span style="color: #26A269">&gt;</span>  
</pre>

now you can type any scheme expression like:

<pre><span style="color: #26A269">&gt;</span> (+ 1 2)
<span style="color: #C01C28">=</span> 3
<span style="color: #26A269">&gt;</span> (&#42; 4 5)
<span style="color: #C01C28">=</span> 20
<span style="color: #26A269">&gt;</span> (show) /* shows all databases */
<span style="color: #C01C28">=</span> ()
<span style="color: #26A269">&gt;</span> (createdatabase &quot;yo&quot;)
<span style="color: #C01C28">=</span> &quot;ok&quot;
<span style="color: #26A269">&gt;</span> (show) /* shows all databases */
<span style="color: #C01C28">=</span> (&quot;yo&quot;)
<span style="color: #26A269">&gt;</span> (show &quot;yo&quot;) /* shows all tables */
<span style="color: #C01C28">=</span> ()
<span style="color: #26A269">&gt;</span> (rebuild) /* optimizes memory layout */
<span style="color: #C01C28">=</span> &quot;124.503µs&quot;
<span style="color: #26A269">&gt;</span> (print (stat)) /* memory usage statistics */
<span style="color: #C01C28">=</span> &quot;Alloc = 0 MiB	TotalAlloc = 1 MiB	Sys = 16 MiB	NumGC = 1&quot;
<span style="color: #26A269">&gt;</span> (loadCSV &quot;yo&quot; &quot;customers&quot; &quot;customers.csv&quot; &quot;;&quot;) /* loads CSV */</pre>

<h2>MySQL Connection</h2>

connect to it via

```bash
mysql -u root -p -P 3307 # password is 'admin'
```

You can try queries like:
```sql
SHOW DATABASES
SHOW TABLES
CREATE TABLE foo(bar string, amount int)
INSERT INTO foo(bar, amount) VALUES ('Man', 4), ('Horse', 6)
SELECT * FROM foo
SELECT SUM(amount) FROM foo
```

If you want to import whole databases from your old MySQL or MariaDB database, do the following:
```
$ mysqldump [PARAMETERS] > dump.sql
$ ./memcp
memcp Copyright (C) 2023   Carl-Philip Hänsch
    This program comes with ABSOLUTELY NO WARRANTY;
    This is free software, and you are welcome to redistribute it
    under certain conditions;
Welcome to memcp
Hello World
MySQL server listening on port 3307 (connect with mysql -P 3307 -u user -p)
listening on http://localhost:4321
> (createdatabase "my_database")
"ok"
> (load_sql "my_database" (stream "dump.sql"))
"1.454ms"
```

If you want to import whole databases from your old PostgreSQL database, do the following:
```
$ pgdump [PARAMETERS] > dump.sql
$ ./memcp
memcp Copyright (C) 2023   Carl-Philip Hänsch
    This program comes with ABSOLUTELY NO WARRANTY;
    This is free software, and you are welcome to redistribute it
    under certain conditions;
Welcome to memcp
Hello World
MySQL server listening on port 3307 (connect with mysql -P 3307 -u user -p)
listening on http://localhost:4321
> (createdatabase "my_database")
"ok"
> (load_psql "my_database" (stream "dump.sql"))
"1.454ms"
```

<h2>REST API</h2>

SQL backend in MySQL syntax mode:
```bash
curl --user root:admin 'http://localhost:4321/sql/system/SHOW%20DATABASES'
curl --user root:admin 'http://localhost:4321/sql/system/SHOW%20TABLES'
curl --user root:admin 'http://localhost:4321/sql/system/SELECT%20*%20FROM%20user'
```

SQL backend in PostgreSQL syntax mode:
```bash
curl --user root:admin 'http://localhost:4321/psql/system/SHOW%20DATABASES'
curl --user root:admin 'http://localhost:4321/psql/system/SHOW%20TABLES'
curl --user root:admin 'http://localhost:4321/psql/system/SELECT%20*%20FROM%20user'
```

You can also define your own endpoints for MemCP and deploy microservices directly in the database (read https://www.memcp.org/wiki/MemCP_for_Microservices).

<hr>

<h1 align="center">Testing 🧪</h1>

MemCP includes a comprehensive test suite to ensure quality and catch regressions.

<h2>Running Tests</h2>

### Automated SQL Test Suite

Run the comprehensive YAML-based SQL test suite:

```bash
# Run tests with default port (4321)
python3 run_sql_tests.py tests/sql_test_spec.yaml

# Run tests with custom port
python3 run_sql_tests.py tests/sql_test_spec.yaml 4322

# The test runner will:
# 1. Automatically start MemCP on the specified port
# 2. Run all test cases defined in the YAML spec
# 3. Report detailed results
# 4. Gracefully shut down MemCP
```

### Manual Testing

You can also test individual components:

```bash
# Test basic connectivity
curl --user root:admin http://localhost:4321/sql/system/SELECT%201%2B2

# Test with custom MemCP instance
./memcp --api-port=4400 --disable-mysql &
python3 test_memcp_api.py  # Legacy Python test script
```

<h2>Writing Tests</h2>

MemCP uses YAML-based test specifications for maintainable, declarative testing.

### Test File Structure

Create test files in the `tests/` directory following this format:

```yaml
# tests/my_feature_test.yaml
metadata:
  version: "1.0"
  description: "Test suite for my new feature"
  database: "test_db"

setup:
  - action: "CREATE DATABASE"
    sql: "CREATE DATABASE test_db"
  - action: "CREATE TABLE"
    sql: "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(100))"

test_cases:
  - name: "Basic functionality test"
    sql: "SELECT 1+1 AS result"
    expect:
      rows: 1
      data:
        - result: 2
        
  - name: "Error handling test"
    sql: "SELECT * FROM non_existent_table" 
    expect:
      error: true
      error_type: "table_not_found"

cleanup:
  - action: "DROP DATABASE"
    sql: "DROP DATABASE test_db"
```

### Test Case Format

Each test case supports:

- **name**: Descriptive test name
- **sql**: SQL query to execute
- **expect**: Expected results
  - **rows**: Expected number of result rows
  - **data**: Expected row data (array of objects)
  - **error**: Set to `true` to expect an error
  - **error_type**: Optional error classification
  - **affected_rows**: For INSERT/UPDATE/DELETE operations

### Running Custom Tests

```bash
python3 run_sql_tests.py tests/my_feature_test.yaml 4323
```

<h2>Continuous Integration</h2>

MemCP includes a pre-commit hook that automatically runs tests before allowing commits:

```bash
# The pre-commit hook will:
# 1. Build the latest MemCP binary
# 2. Run the complete test suite
# 3. Block commits if tests fail
# 4. Provide clear feedback on failures

# To test the hook manually:
.git/hooks/pre-commit

# Skip hook for emergency commits (not recommended):
git commit --no-verify
```

<hr>

<h1 align="center">Deployment 🚀</h1>

<h2>Production Deployment</h2>

### Systemd Service

Create `/etc/systemd/system/memcp.service`:

```ini
[Unit]
Description=MemCP In-Memory Database
After=network.target

[Service]
Type=simple
User=memcp
Group=memcp
WorkingDirectory=/opt/memcp
ExecStart=/opt/memcp/memcp -data /var/lib/memcp --api-port=4321 --mysql-port=3307
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable memcp
sudo systemctl start memcp
sudo systemctl status memcp
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  memcp:
    build: .
    ports:
      - "4321:4321"  # HTTP API
      - "3307:3307"  # MySQL Protocol
    volumes:
      - memcp_data:/data
    environment:
      - PORT=4321
      - MYSQL_PORT=3307
    restart: unless-stopped
    
volumes:
  memcp_data:
```

### Kubernetes Deployment

```yaml
# memcp-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: memcp
spec:
  replicas: 1
  selector:
    matchLabels:
      app: memcp
  template:
    metadata:
      labels:
        app: memcp
    spec:
      containers:
      - name: memcp
        image: memcp:latest
        ports:
        - containerPort: 4321
        - containerPort: 3307
        env:
        - name: PORT
          value: "4321"
        - name: MYSQL_PORT
          value: "3307"
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: memcp-data
```

### Load Balancing

For high-availability setups, use a load balancer in front of multiple MemCP instances:

```nginx
# nginx.conf
upstream memcp_api {
    server 127.0.0.1:4321;
    server 127.0.0.1:4322;
    server 127.0.0.1:4323;
}

server {
    listen 80;
    location / {
        proxy_pass http://memcp_api;
        proxy_set_header Authorization $http_authorization;
    }
}
```

<hr>
<h1 align="center">Securing the database from external access 🔒</h1>
The standard username/password is root/admin. To change that, type the following into scheme console:

```
(eval (parse_sql "" "ALTER USER root IDENTIFIED BY 'new_password'"))
(eval (parse_sql "" "CREATE USER user2 IDENTIFIED BY 'new_password'"))
```

<hr>

<h1 align="center">Example REST API App 📕</h1>

You can take a look at https://github.com/launix-de/rdfop which is a RDF hypertext processor based on MemCP

<hr>

<h1 align="center">Contributing 🌿</h1>

<p align="center"> We welcome contributions to MemCP! Please follow these guidelines to ensure quality and maintainability. </p>

## 🚨 **Test Requirements - MANDATORY** 🚨

**ALL CONTRIBUTIONS MUST INCLUDE COMPREHENSIVE TEST CASES.** This is not optional.

### For New Features:
- Create a new YAML test file in `tests/feature_name_test.yaml`
- Cover all major code paths and edge cases
- Include both success and error scenarios
- Document expected behavior clearly

### For Bug Fixes:
- Add regression tests that reproduce the original bug
- Verify the fix resolves the issue
- Ensure no existing functionality breaks

### For API Changes:
- Update existing tests to match new behavior
- Add tests for new endpoints or parameters
- Maintain backward compatibility where possible

## Contributing Process

1. **Fork** the repository and create a feature branch:
   ```bash
   git checkout -b feature/amazing-new-feature
   ```

2. **Implement** your changes with proper error handling and documentation

3. **Write comprehensive tests** following our YAML test format:
   ```bash
   # Create tests/your_feature_test.yaml
   # Run your tests locally
   python3 run_sql_tests.py tests/your_feature_test.yaml 4500
   ```

4. **Ensure all tests pass**:
   ```bash
   # Run the full test suite
   python3 run_sql_tests.py tests/sql_test_spec.yaml 4400
   
   # Test the pre-commit hook
   .git/hooks/pre-commit
   ```

5. **Commit with descriptive messages**:
   ```bash
   git add .
   git commit -m "Add feature X with comprehensive tests
   
   - Implements new SQL function Y
   - Adds 15 test cases covering edge cases
   - Fixes issue #123"
   ```

6. **Push and create a pull request**:
   ```bash
   git push origin feature/amazing-new-feature
   ```

## Code Quality Standards

- **Test Coverage**: Every line of new code must be tested
- **Error Handling**: Implement proper error messages and HTTP status codes
- **Documentation**: Update README.md for user-facing changes
- **Performance**: Consider memory usage and query performance impact
- **Security**: Follow secure coding practices, never expose credentials

## Pre-commit Hook

The pre-commit hook will automatically:
- ✅ Build your changes
- ✅ Run the complete test suite  
- ❌ **Block commits if ANY tests fail**

This ensures the main branch always remains stable.

## Review Process

Pull requests will be reviewed for:

1. **Test Quality**: Comprehensive coverage of new functionality
2. **Code Quality**: Clean, readable, maintainable code
3. **Performance**: No significant performance regressions
4. **Documentation**: Clear explanations of changes
5. **Compatibility**: Backward compatibility preservation

## What Happens If You Don't Include Tests?

**Your pull request will be rejected immediately.** No exceptions.

We maintain a high-quality codebase through rigorous testing. This benefits everyone by:
- Preventing regressions
- Ensuring feature reliability  
- Making debugging easier
- Providing executable documentation
- Enabling confident refactoring

<p align="center"> <strong>Remember: Good tests are as important as good code!</strong> </p>

<hr>

<h1 align="center">How it Works? ❓</h1>

- MemCP structures its data into databases and tables
- Every table has multiple columns and multiple data shards
- Every data shard stores ~64,000 items and is meant to be processed in ~100ms
- Parallelization is done over shards
- Every shard consists of two parts: main storage and delta storage
- main storage is column-based, fixed-size and is compressed
- Delta storage is a list of row-based insertions and deletions that is overlaid over a main storage
- `(rebuild)` will merge all main+delta storages into new compressed main storages with empty delta storages
- every dataset has a shard-local so-called `recordId` to re-identify a dataset


<h1 align="center">Available column compression formats 📃</h1>

- uncompressed & untyped: versatile storage with JSON-esque freedom
- bit-size reduced integer storage with offset: savings of 80% and more for small integers
- integer sequences: >99% compression ratio possible with ascending IDs
- string-storage: more compact than C strings, cache-friendly 
- string-dictionary: >90% memory savings for repeating strings like (male, female, male, male, male)
- float storage
- sparse storage: efficient with lots of NULL values

the best suitable compression technique for a column is detected for a column <b>automatically</b>

<hr>

<h1 align="center">Frequently Asked Questions 🤔</h1>

### What is an in-memory database?
Unlike traditional databases, which store data on disks, in-memory databases (IMDBs) keep data in RAM. This results in much faster access times.

### Why it is used?
An in-memory database (IMDB) stores and retrieves data primarily in a computer's RAM, enabling exceptionally fast data processing and retrieval, making it suitable for real-time applications requiring rapid access to data.

### What are the benefits of columnar storage?
With columnar storage, data is much more homogeneous than in row-based storage. This enables a technique called "column compression" where compression ratios of around 1:5 (i.e. 80% savings) can be achieved just by a different data representation. This reduces the amount of cache lines that must be transferred from main memory to CPU and thus increases performance, reduces power consumption and decreases latency.

Also, columnar storages are a better fit for analytical queries where only a few out of possibly hundreds of columns are processed in the SQL query. An example of an analytical query is calculating the sum of revenue over a timespan from a lot of data points.

### Can in-memory databases be used for my web project?
Yes. MemCP is meant as a drop-in replacement for MySQL and will make your application run faster.

### Why does MemCP consume less RAM than MySQL even though MySQL is a hard disk-based database
In order to run fast, MySQL already has to cache all data in RAM. However, MySQL is not capable of compression, so it will consume about 5x the amount of RAM compared to MemCP for the same size of data cache.

### Isn't it dangerous to keep all data in RAM? What happens during a crash?
MemCP of course supports some kind of hard disk persistency. The difference to a hard-disk-based database is that in MemCP you can choose how much IO bandwidth you want to sacrifice to achieve full crash safety. In other words: Your accounting data can still be secured with per-transaction write barriers while you can increase the write performance for sensor data by loosening persistency guarantees.

### What happens if memory is full?
Usually, the net amount of data in databases is very low. You will be amazed, at how much data fits into your RAM when properly compressed. If that still exceeds the memory of your machine, just remember how slow it would be on the hard disk. Upgrade your RAM if you don't want to be on your swap partition. When you really go big data, a shared memory cluster is the way for you to go.


### What's the current development status of MemCP?
We are still in the early alpha phase. MemCP already supports some basic SQL statements but it is not production-ready yet. The best way to use MemCP in a productive environment is over the internal scheme scripting language where you can hand-craft efficient query plans. Contribution to the SQL compiler is of course welcome.

### What are MemCP REST services?
Normally, REST applications are implemented in any programming language, make a connection to an SQL server and do their queries. This induces IO overhead for that additional network layer between application and database and for the string-print-send-receive-parse pipeline. With MemCP, you can script MemCP to open a REST server and offer your REST routes directly in the process space of the database. You can prepare SQL statements which can be directly invoked inside the database. And don't be afraid of crashes: a crash in MemCPs scheme scripts will never bring down the whole database process.

<hr>

<h1 align="center">Further Reading 📚</h1>

- [VLDB Research Paper](https://www.vldb.org/pvldb/vol13/p2649-boncz.pdf)
- [LNI Proceedings Paper](https://cs.emis.de/LNI/Proceedings/Proceedings241/383.pdf)
- [TU Dresden Research Paper](https://wwwdb.inf.tu-dresden.de/wp-content/uploads/T_2014_Master_Patrick_Damme.pdf)
- [Large Graph Algorithms](https://www.dcs.bbk.ac.uk/~dell/teaching/cc/paper/sigmod10/p135-malewicz.pdf)

- [Balancing OLAP and OLTP Workflows](https://launix.de/launix/how-to-balance-a-database-between-olap-and-oltp-workflows/)
- [Designing Programming Languages for Distributed Systems](https://launix.de/launix/designing-a-programming-language-for-distributed-systems-and-highly-parallel-algorithms/)
- [Columnar Storage Interface in Golang](https://launix.de/launix/on-designing-an-interface-for-columnar-in-memory-storage-in-golang/)
- [Impact of In-Memory Compression on Performance](https://launix.de/launix/how-in-memory-compression-affects-performance/)
- [Memory-Efficient Indices for In-Memory Storages](https://launix.de/launix/memory-efficient-indices-for-in-memory-storages/)
- [Compressing Null Values in Bit-Compressed Integer Storages](https://launix.de/launix/on-compressing-null-values-in-bit-compressed-integer-storages/)
- [Improving Golang HTTP Server Performance](https://launix.de/launix/when-the-benchmark-is-too-slow-golang-http-server-performance/)
- [Benchmarking SQL Databases](https://launix.de/launix/how-to-benchmark-a-sql-database/)
- [Writing a SQL Parser in Scheme](https://launix.de/launix/writing-a-sql-parser-in-scheme/)
- [Accessing memcp via Scheme](https://launix.de/launix/accessing-memcp-via-scheme/)
- [First SQL Query in memcp](https://launix.de/launix/memcp-first-sql-query-is-correctly-executed/)
- [Sequence Compression in In-Memory Database](https://launix.de/launix/sequence-compression-in-in-memory-database-yields-99-memory-savings-and-a-total-of-13/)
- [Storing Data Smaller Than One Bit](https://launix.de/launix/storing-a-bit-smaller-than-in-one-bit/)
- [memcp Announcement Video](https://www.youtube.com/watch?v=DWg4nx4KVLo)
- https://wwwdb.inf.tu-dresden.de/research-projects/eris/
- https://hyper-db.de/
- All Information about MemCP can be found under [memcp.org](https://memcp.org)
