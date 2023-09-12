<h1 align="center">memcp: A High-Performance, Open-Source Columnar In-Memory Database </h1>
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

### What is memcp?
memcp is an open-source, high-performance, columnar in-memory database that can handle both OLAP and OLTP workloads. It provides an alternative to proprietary analytical databases and aims to bring the benefits of columnar storage to the open-source world.

###
memcp is written in Golang and is designed to be portable and extensible, allowing developers to embed the database into their applications with ease. It is also designed with a focus on scalability and performance, making it a suitable choice for distributed applications.

</div>

<br>

### Features
- <b>fast:</b> MemCP is built with parallelization in mind. The parallelization pattern is made for minimal overhead.
- <b>efficient:</b> Compression is 1:5 (80% memory saving) compared to MySQL/MariaDB
- <b>modern:</b> MemCP is built for modern hardware with caches, NUMA memory, multicore CPUs, NVMe SSDs
- <b>versatile:</b> Use it in big mainframes to gain analytical performance, use it in embedded systems to conserve flash lifetime
- Columnar storage: Stores data column-wise instead of row-wise, which allows for better compression, faster query execution, and more efficient use of memory.
- In-memory database: Stores all data in memory, which allows for extremely fast query execution.
- Build fast REST APIs directly in the database (they are faster because there is no network connection / SQL layer in between)
- OLAP and OLTP support: Can handle both online analytical processing (OLAP) and online transaction processing (OLTP) workloads.
- Bit-packing and dictionary encoding: Uses bit-packing and dictionary encoding to achieve higher compression ratios for integer and string data types, respectively.
- Delta storage: Maintains separate delta storage for updates and deletes, which allows for more efficient handling of OLTP workloads.
- Scalability: Designed to scale on a single node with huge NUMA memory
- Adjustable persistency: Decide whether you want to persist a table or not or to just keep snapshots of a period of time

<hr>

<h1 align="center"> Screenshots </h1>


<div align="center">

<img src="https://i.ibb.co/fCWvndp/Add-a-subheading.png" alt="Benchmark" border="0">

<br>

<img src="https://i.ibb.co/s9npgmq/Add-a-subheading-1.png" alt="mysql client connecting to memcp" border="0">

</div>

<hr>

<h1 align="center">Getting Started 🚶</h1>

Compile the project with

```
make
```

Run the engine with

```
./memcp
```

connect to it via

```
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
<hr>

<h1 align="center">Example REST API App 📕</h1>

```
./memcp apps/bayesian.scm
```

now you can use the Bayesian text classifier under http://localhost:4321/bayes/ as a REST service

```
curl 'http://localhost:4321/bayes/i am a booking text?account=4001' # will learn the text to be account=4001
curl 'http://localhost:4321/bayes/i am a booking text?classify=account' # will return 4001
```

<hr>

<h1 align="center">Contributing 🌿</h1>

<p align="center"> We welcome contributions to memcp. If you would like to contribute, please follow these steps:, </p>

- Fork the repository and create a new branch for your changes.
- Make your changes and commit them to your branch.
- Push your branch to your fork and create a pull request.

<p align="center"> Before submitting a pull request, please make sure that your changes pass the existing tests and add new tests if necessary. </p>

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
With columnar storage, data is much more homogeneous than in a row-based storage. This enables a technique called "column compression" where compression ratios of around 1:5 (i.e. 80% savings) can be achieved just by a different data representation. This reduces the amount of cache lines that have to be transferred from main memory to CPU and thus increases performance, reduce power consumption and decreases latency.

Also, columnar storages are better fit for analytical queries where only few out of possibly hundrets of columns are processed in the SQL query. An example for a analytical query is calculating the sum of revenue over a timespan from a lots of datapoints.

### Can in-memory databases be used for my web project?
Yes. MemCP is ment as a drop-in replacement for MySQL and will make your application run faster.

### Why does MemCPu consume less RAM than MySQL even though MySQL is a hard disk based database
In order to run fast, MySQL already has to cache all data in RAM. However, MySQL is not capable of compression, so it will consume about 5x the amount of RAM compared to MemCP for the same size of data cache.

### Isn't it dangerous to keep all data in RAM? What happens during a crash?
MemCP of course supports some kind of hard disk persistency. The difference to a hard-disk based database is that in MemCP you can choose who much IO bandwith you want to sacrifice to achieve full crash-safety. In other words: Your accounting data can still be secured with per-transaction write barriers while you can increase the write performance for sensor data by loosening persistency guarantees.

### What's the current development status of MemCP?
We are still in the early alpha phase. MemCP already supports some basic SQL statements but it is not production-ready yet. The best way to use MemCP in a productive environment is over the internal scheme scripting language where you can hand-craft efficient query plans. Contribution to the SQL compiler is of course welcome.

### What are MemCP REST services?
Normally, REST applications are implemented in any programming language, make a connection to a SQL server and do their queries. This induces IO overhead for that additional network layer between application and database and for the string-print-send-receive-parse pipeline. With MemCP, you can script MemCP to open a REST server and offer your REST routes directly in the process space of the database. You can prepare SQL statements which can be directly invoked inside the database. And don't be afraid of crashes: a crash in MemCPs scheme scripts will never bring down the whole database process.

<hr>

<h1 align="center">Further Reading 📚</h1>

- [VLDB Research Paper](https://www.vldb.org/pvldb/vol13/p2649-boncz.pdf)
- [LNI Proceedings Paper](https://cs.emis.de/LNI/Proceedings/Proceedings241/383.pdf)
- [TU Dresden Research Paper](https://wwwdb.inf.tu-dresden.de/wp-content/uploads/T_2014_Master_Patrick_Damme.pdf)
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
