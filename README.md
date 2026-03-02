# MemCP - Ultra-Fast In-Memory Database 🚀

**Ready to supercharge your applications?** MemCP is a blazing-fast, MySQL-compatible database that runs entirely in-memory, delivering **unprecedented performance** for modern web applications and APIs.

## Why Switch from MySQL? 💡

### ⚡ **10-100x Faster Than Traditional Databases**
- **Zero disk I/O latency** - everything runs in RAM
- **Sub-millisecond query response times**
- **Ultra-fast REST APIs** with built-in HTTP server
- **No connection overhead** - direct in-process access

### 🔌 **Drop-in MySQL Compatibility**
```sql
-- Your existing MySQL queries work immediately
CREATE TABLE users (id INT, name VARCHAR(100), email VARCHAR(255));
INSERT INTO users VALUES (1, 'Alice', 'alice@example.com');
SELECT * FROM users WHERE id = 1;
```

### 🌐 **Built-in REST API Server**
```bash
# Start MemCP with REST API
./memcp --api-port=4321 lib/main.scm

# Query via HTTP instantly
curl -X POST http://localhost:4321/sql/mydb \
  -d "SELECT * FROM users" \
  -H "Authorization: Basic cm9vdDphZG1pbg=="
```

**API Endpoints:**
- `/sql/<database>` — MySQL-dialect SQL
- `/psql/<database>` — PostgreSQL-dialect SQL
- `/rdf/<database>` — SPARQL queries
- `/rdf/<database>/load_ttl` — load RDF/Turtle data
- `/dashboard` — WebSocket admin dashboard

### 📊 **Perfect for Modern Workloads**
- **Microservices** - Embedded database per service
- **APIs and Web Apps** - Ultra-low latency responses  
- **Real-time Analytics** - Process data as fast as it arrives
- **Development & Testing** - Instant setup, no configuration

## Architecture & Languages 🏗️

MemCP combines the best of multiple worlds with a carefully chosen tech stack:

### **Go (Storage Engine & Core)**
- **High-performance storage engine** built in Go
- **Concurrent request handling** with goroutines
- **Memory-efficient data structures**
- **Cross-platform compatibility**

### **Scheme (SQL Parser & Compiler)**
- **Advanced SQL parser** written in Scheme
- **Query optimization and compilation**
- **Extensible language for complex transformations**
- **Functional programming advantages for parsing**

### **Flexible Scripting Support**
- **Command-line argument support** for automation
- **Dynamic query generation** and processing
- **Easy integration** with existing workflows

## Key Advantages 🎯

### **🔥 Ultra-Fast REST APIs**
Traditional setup: `Client → HTTP Server → Database Connection → Disk I/O`
**MemCP**: `Client → HTTP Server → In-Memory Data` ✨

```javascript
// Response times you'll see
MySQL (with network + disk):  10-50ms
MemCP (in-memory):           0.1-1ms  // 50x faster!
```

### **⚡ Docker**
```bash
docker pull carli2/memcp
docker run -it -p 4321:4321 -p 3307:3307 carli2/memcp
```

### **🧠 Smart Memory Management**
- **Automatic data optimization** for memory usage
- **Configurable memory limits**
- **Efficient garbage collection**
- **Data persistence options** when needed

### **🔧 Developer-Friendly**
- **Comprehensive test suite** with 2470+ SQL tests across 100+ test suites
- **YAML-based testing framework**
- **Extensive error handling and validation**
- **Built-in performance monitoring**

## Quick Start 🚀

```bash
# 1. Build MemCP
go mod download
make

# 2. Start with REST API
./memcp --api-port=4321 --mysql-port=3307 lib/main.scm

# Run as a background daemon (use --no-repl to avoid exiting when stdin closes)
./memcp --no-repl --api-port=4321 --mysql-port=3307 lib/main.scm &

# 3. Create your first database
curl -X POST http://localhost:4321/sql/system \
  -d "CREATE DATABASE myapp" \
  -u root:admin

# 4. Start building lightning-fast apps!
curl -X POST http://localhost:4321/sql/myapp \
  -d "CREATE TABLE products (id INT, name VARCHAR(100), price DECIMAL(10,2))" \
  -u root:admin

```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--api-port=PORT` | `4321` | HTTP API listen port |
| `--mysql-port=PORT` | `3307` | MySQL protocol listen port |
| `--mysql-socket=PATH` | `/tmp/memcp.sock` | MySQL Unix socket path |
| `--root-password=PASSWORD` | `admin` | Initial root password (first run only) |
| `--disable-api` | — | Disable HTTP API server |
| `--disable-mysql` | — | Disable MySQL protocol server |
| `--no-repl` | — | Disable interactive REPL (required for daemon/background use) |
| `-data DIR` | `./data` | Data directory |

### Authentication
- Default credentials: `root` / `admin`.
- Set the initial root password via CLI: `--root-password=supersecret` at the first run (on a fresh -data folder), or via Docker env `ROOT_PASSWORD`.
- Docker Compose example:
```yaml
services:
  memcp:
    image: carli2/memcp:latest
    environment:
      - ROOT_PASSWORD=supersecret
      - PARAMS=--api-port=4321
    ports:
      - "4321:4321"  # HTTP API
      - "3307:3307"  # MySQL protocol
    volumes:
      - memcp_data:/data
volumes:
  memcp_data: {}
```
- Change the credentials with:
```
curl -X POST http://localhost:4321/sql/system \
  -d "ALTER USER root IDENTIFIED BY 'supersecret'" \
  -u root:admin
```

## Performance Comparison 📈

| Operation | MySQL (SSD) | MySQL (Memory) | **MemCP** |
|-----------|-------------|----------------|-----------|
| Simple SELECT | 5-15ms | 1-3ms | **0.1ms** |
| Complex JOIN | 50-200ms | 10-50ms | **1-5ms** |
| INSERT/UPDATE | 10-30ms | 2-8ms | **0.2ms** |
| REST API Call | 20-100ms | 10-60ms | **1-10ms** |

*Benchmarks run on standard hardware with 1000+ concurrent requests*

## Use Cases 💼

- **🎮 Gaming Backends** - Real-time leaderboards and player data
- **💰 Financial APIs** - High-frequency trading and analytics  
- **📱 Mobile Apps** - Ultra-responsive user experiences
- **🛒 E-commerce** - Product catalogs and inventory management
- **📊 Analytics Dashboards** - Real-time data visualization
- **🧪 Development & Testing** - Instant database provisioning

## Contributing 🤝

**We'd love your help making MemCP even better!** 

### 🌟 **Why Contribute?**
- Work with **cutting-edge database technology**
- Learn **Go, Scheme, and database internals**
- Impact **thousands of developers** worldwide
- Build **ultra-high-performance systems**

### 🛠️ **Easy Ways to Contribute**
- **📝 Add test cases** - Expand our comprehensive test suite
- **🐛 Fix bugs** - Help us squash issues and improve stability  
- **⚡ Performance optimization** - Make fast even faster
- **📚 Documentation** - Help other developers get started
- **🔧 New features** - SQL functions, operators, and capabilities

### 🚀 **Getting Started**
```bash
# 1. Fork the repository
# 2. Clone your fork
git clone https://github.com/launix-de/memcp.git

# 3. Set up development environment
cd memcp
go build -o memcp

# 4. Run the test suite (starts its own server automatically)
python3 run_sql_tests.py tests/01_basic_sql.yaml

# 5. Make your changes and add tests
# 6. Submit a pull request!
```

### 🎯 **Current Contribution Opportunities**
- **Vector database features** - Advanced similarity search
- **Additional SQL functions** - String, math, and date functions
- **Performance benchmarking** - Automated performance testing
- **Driver development** - Language-specific database drivers
- **Documentation examples** - Real-world usage scenarios

## Testing 🧪

MemCP includes a comprehensive test framework:

```bash
# Run all tests
make test

# Or if you want to contribute, deploy this as a Pre-commit hook:
cp git-pre-commit .git/hooks/pre-commit

# Run specific test suites (starts its own server automatically)
python3 run_sql_tests.py tests/01_basic_sql.yaml      # Basic operations
python3 run_sql_tests.py tests/02_functions.yaml      # SQL functions
python3 run_sql_tests.py tests/07_error_cases.yaml    # Error handling

# Connect to an already-running instance (skip startup)
python3 run_sql_tests.py tests/01_basic_sql.yaml 4321 --connect-only
```

## Performance Testing 📊

MemCP includes an auto-calibrating performance test framework that adapts to your machine.

### Running Performance Tests

```bash
# Run perf tests (uses calibrated baselines)
PERF_TEST=1 make test

# Calibrate for your machine (run ~10 times to reach target time range)
PERF_TEST=1 PERF_CALIBRATE=1 make test

# Freeze row counts for bisecting performance regressions
PERF_TEST=1 PERF_NORECALIBRATE=1 make test

# Show query plans for each test
PERF_TEST=1 PERF_EXPLAIN=1 make test
```

### How Calibration Works

1. **Initial run** starts with 10,000 rows per test
2. Each calibration run **scales row counts by 30%** up/down
3. Target is **10-20 seconds** query time per test
4. Baselines are stored in `.perf_baseline.json`
5. After ~10 runs, row counts stabilize in the target range

### Output Format

```
✅ Perf: COUNT (7.9ms / 8700ms, 20,000 rows, 0.39µs/row, 11.4MB heap)
         │       │        │           │        │           └─ Heap memory after insert
         │       │        │           │        └─ Time per row
         │       │        │           └─ Calibrated row count
         │       │        └─ Threshold (from baseline × 1.1)
         │       └─ Actual query time
         └─ Test name
```

### Performance Debugging Cookbook

**Detecting a performance regression:**
```bash
# 1. Freeze baselines to use consistent row counts
PERF_TEST=1 PERF_NORECALIBRATE=1 make test

# 2. If a test fails threshold, you have a regression
```

**Bisecting a performance bug:**
```bash
# 1. Checkout the known-good commit, run calibration
git checkout good-commit
PERF_TEST=1 PERF_CALIBRATE=1 make test  # run 10x to calibrate

# 2. Save the baseline
cp .perf_baseline.json .perf_baseline_good.json

# 3. Bisect with frozen row counts
git bisect start
git bisect bad HEAD
git bisect good good-commit
git bisect run bash -c 'PERF_TEST=1 PERF_NORECALIBRATE=1 make test'
```

**Analyzing slow queries:**
```bash
# Show query plans to understand execution
PERF_TEST=1 PERF_EXPLAIN=1 make test
```

### Environment Variables

| Variable | Values | Description |
|----------|--------|-------------|
| `PERF_TEST` | `0`/`1` | Enable performance tests |
| `PERF_CALIBRATE` | `0`/`1` | Update baselines with new times |
| `PERF_NORECALIBRATE` | `0`/`1` | Freeze row counts (for bisecting) |
| `PERF_EXPLAIN` | `0`/`1` | Show query plans |

## Remote Storage Backends 🗄️

MemCP supports storing databases on remote storage backends instead of the local filesystem. To configure a remote backend, create a JSON configuration file in the data folder instead of a directory.

### S3 / MinIO Storage

Store your database on Amazon S3 or any S3-compatible storage (MinIO, Ceph RGW, etc.).

**Configuration file** (`data/mydb.json`):
```json
{
  "backend": "s3",
  "access_key_id": "your-access-key",
  "secret_access_key": "your-secret-key",
  "region": "us-east-1",
  "bucket": "memcp-data",
  "prefix": "databases"
}
```

**For MinIO or self-hosted S3-compatible storage:**
```json
{
  "backend": "s3",
  "access_key_id": "minioadmin",
  "secret_access_key": "minioadmin",
  "endpoint": "http://localhost:9000",
  "bucket": "memcp",
  "prefix": "data",
  "force_path_style": true
}
```

**Quick MinIO setup for testing:**
```bash
# Start MinIO with Docker
docker run -d --name minio \
  -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Create a bucket (via MinIO Console at http://localhost:9001)
# Or via mc CLI:
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb local/memcp
```

### Ceph/RADOS Storage

Store your database directly on Ceph RADOS for high-performance distributed storage.

**Why is Ceph optional?** The Ceph backend uses CGO to link against `librados` (the Ceph client library). This requires the C headers and library to be installed at compile time and the shared library at runtime. To keep the default build simple and portable, Ceph support is behind a build tag.

```bash
# Install Ceph development libraries (Ubuntu/Debian)
sudo apt-get install librados-dev

# Build MemCP with Ceph support
make ceph
# or: go build -tags=ceph
```

**Configuration file** (`data/mydb.json`):
```json
{
  "backend": "ceph",
  "username": "client.memcp",
  "cluster": "ceph",
  "pool": "memcp",
  "prefix": "databases"
}
```

**Optional fields:**
- `conf_file`: Path to ceph.conf (defaults to `/etc/ceph/ceph.conf`)

**Setting up a Ceph development cluster with vstart.sh:**
```bash
# Clone Ceph source
git clone https://github.com/ceph/ceph.git
cd ceph

# Install dependencies and build (only vstart target needed)
./install-deps.sh
pip install cython setuptools
./do_cmake.sh
cd build && ninja vstart

# Start a development cluster
cd ..
MON=1 OSD=3 MDS=0 MGR=1 ./build/bin/vstart.sh -d -n -x

# Create a pool for MemCP
./build/bin/ceph osd pool create memcp 32

# Create a user for MemCP (optional, can also use client.admin)
./build/bin/ceph auth get-or-create client.memcp \
  mon 'allow r' \
  osd 'allow rwx pool=memcp' \
  -o ceph.client.memcp.keyring
```

**Environment variables for vstart cluster:**
```bash
export CEPH_CONF=/path/to/ceph/build/ceph.conf
export CEPH_KEYRING=/path/to/ceph/build/keyring
```

### Backend Configuration Reference

| Field | Backend | Description |
|-------|---------|-------------|
| `backend` | all | Backend type: `"s3"` or `"ceph"` |
| `prefix` | all | Object key prefix for database objects |
| `access_key_id` | S3 | AWS or S3-compatible access key |
| `secret_access_key` | S3 | AWS or S3-compatible secret key |
| `region` | S3 | AWS region (e.g., `"us-east-1"`) |
| `endpoint` | S3 | Custom endpoint URL (for MinIO, etc.) |
| `bucket` | S3 | S3 bucket name |
| `force_path_style` | S3 | Use path-style URLs (required for MinIO) |
| `username` | Ceph | Ceph user (e.g., `"client.admin"`) |
| `cluster` | Ceph | Cluster name (usually `"ceph"`) |
| `conf_file` | Ceph | Path to ceph.conf (optional) |
| `pool` | Ceph | RADOS pool name |

## License 📄

MemCP is open source software. See the LICENSE file for details.

---

**Ready to experience database performance like never before?**
[Get Started](#quick-start) • [Contribute](#contributing) • [Join our Community](https://github.com/launix-de/memcp/discussions)

*MemCP: Because your applications deserve better than "good enough" performance.* ⚡
