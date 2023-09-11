<h1 align="center">memcp: A High-Performance, Open-Source Columnar In-Memory Database </h1>

<div align="center">

![memcp >](assets/memcp-logo.svg?raw=true)

memcp is an open-source, high-performance, columnar in-memory database that can handle both OLAP and OLTP workloads. It provides an alternative to proprietary analytical databases and aims to bring the benefits of columnar storage to the open-source world.

memcp is open source and released under the GPLv3 License. It is an open source alternative to proprietary columnar in-memory databases, providing a powerful, reliable, and fast database solution for applications that require high performance and scalability.

memcp is written in golang and is designed to be portable and extensible, allowing developers to embed the database into their applications with ease. It is also designed with a focus on scalability and performance, making it a suitable choice for distributed applications.

</div>

Features
- Columnar storage: Stores data column-wise instead of row-wise, which allows for better compression, faster query execution, and more efficient use of memory.
- In-memory database: Stores all data in memory, which allows for extremely fast query execution.
- OLAP and OLTP support: Can handle both online analytical processing (OLAP) and online transaction processing (OLTP) workloads.
- Bit-packing and dictionary encoding: Uses bit-packing and dictionary encoding to achieve higher compression ratios for integer and string data types, respectively.
- Delta storage: Maintains a separate delta storage for updates and deletes, which allows for more efficient handling of OLTP workloads.
- Scalability: Designed to scale on a single node with huge NUMA memory
- Adjustable persistency: Decide whether you want to persist a table or not or to just keep snapshots of a period of time

<hr>

<h1 align="center"> Screenshots </h1>

<div align="center">

![image](https://github.com/rohankishore/memcp/assets/109947257/f72c68a8-2694-410f-a993-f6fc885776e8)


</div>

<hr>

<h1 align="center"> Getting Started </h1>

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
```
SHOW DATABASES
SHOW TABLES
CREATE TABLE foo(bar string, amount int)
INSERT INTO foo(bar, amount) VALUES ('Man', 4), ('Horse', 6)
SELECT * FROM foo
SELECT SUM(amount) FROM foo
```
<hr>

<h1 align="center"> Example REST API app </h1>

```
./memcp apps/bayesian.scm
```

now you can use the bayesian classifier under http://localhost:4321/bayes/ as a REST service

<hr>

<h1 align="center"> Contributing </h1>

<p align="center"> We welcome contributions to memcp. If you would like to contribute, please follow these steps:, </p>

- Fork the repository and create a new branch for your changes.
- Make your changes and commit them to your branch.
- Push your branch to your fork and create a pull request.

<p align="center"> Before submitting a pull request, please make sure that your changes pass the existing tests and add new tests if necessary. </p>

<hr>

<h1 align="center"> How it works </h1>

- MemCP supports multiple databases that can have multiple tables
- Every table has multiple columns and multiple data shards
- Every data shard has ~64,000 items and is ment to be processed in ~100ms
- Parallelization is done over shards
- every shard consists of two parts: main storage and delta storage
- main storage is column-based, fixed-size and is compressed
- delta storage is a list of row-based insertions and deletions that is overlaid over a main storage
- (rebuild) will merge all main+delta storages into new compressed main storages with empty delta storages
- every dataset has a shard-local so-called recordId

<h1 align="center"> Available column compression formats </h1>

- uncompressed
- bit-size reduced integer storage with offset
- integer sequences (based on 3x integer storage)
- string-storage
- string-dictionary (based on integers)
- float storage
- sparse storage (efficient with lots of NULL values)
- prefix storage (optimizes strings that start with the same substring over and over)

<hr>


# Further Reading

- https://www.vldb.org/pvldb/vol13/p2649-boncz.pdf
- https://cs.emis.de/LNI/Proceedings/Proceedings241/383.pdf
- https://wwwdb.inf.tu-dresden.de/wp-content/uploads/T_2014_Master_Patrick_Damme.pdf
- https://launix.de/launix/how-to-balance-a-database-between-olap-and-oltp-workflows/
- https://launix.de/launix/designing-a-programming-language-for-distributed-systems-and-highly-parallel-algorithms/
- https://launix.de/launix/on-designing-an-interface-for-columnar-in-memory-storage-in-golang/
- https://launix.de/launix/how-in-memory-compression-affects-performance/
- https://launix.de/launix/memory-efficient-indices-for-in-memory-storages/
- https://launix.de/launix/on-compressing-null-values-in-bit-compressed-integer-storages/
- https://launix.de/launix/when-the-benchmark-is-too-slow-golang-http-server-performance/
- https://launix.de/launix/how-to-benchmark-a-sql-database/
- https://launix.de/launix/writing-a-sql-parser-in-scheme/
- https://launix.de/launix/accessing-memcp-via-scheme/
- https://launix.de/launix/memcp-first-sql-query-is-correctly-executed/
- https://launix.de/launix/sequence-compression-in-in-memory-database-yields-99-memory-savings-and-a-total-of-13/
- https://launix.de/launix/storing-a-bit-smaller-than-in-one-bit/
- https://launix.de/launix/announcement-memcp-gets-adaptible-consistency-layer/
- https://www.youtube.com/watch?v=DWg4nx4KVLo
