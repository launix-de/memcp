# memcp: A High-Performance, Open-Source Columnar In-Memory Database

![memcp >](memcp-logo.svg?raw=true)

memcp is an open-source, high-performance, columnar in-memory database that can handle both OLAP and OLTP workloads. It provides an alternative to proprietary analytical databases and aims to bring the benefits of columnar storage to the open-source world.

memcp is open source and released under the GPLv3 License. It is an open source alternative to proprietary columnar in-memory databases, providing a powerful, reliable, and fast database solution for applications that require high performance and scalability.

memcp is written in golang and is designed to be portable and extensible, allowing developers to embed the database into their applications with ease. It is also designed with a focus on scalability and performance, making it a suitable choice for distributed applications.

Features
- Columnar storage: Stores data column-wise instead of row-wise, which allows for better compression, faster query execution, and more efficient use of memory.
- In-memory database: Stores all data in memory, which allows for extremely fast query execution.
- OLAP and OLTP support: Can handle both online analytical processing (OLAP) and online transaction processing (OLTP) workloads.
- Bit-packing and dictionary encoding: Uses bit-packing and dictionary encoding to achieve higher compression ratios for integer and string data types, respectively.
- Delta storage: Maintains a separate delta storage for updates and deletes, which allows for more efficient handling of OLTP workloads.
- Scalability: Designed to scale horizontally across multiple nodes to handle large data volumes and high query throughput.

# Getting Started

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
CREATE TABLE foo(bar string, amount int)
INSERT INTO foo(bar, amount) VALUES ('Man', 4), ('Horse', 6)
SELECT * FROM foo
SELECT SUM(amount) FROM foo
```

# Contributing

We welcome contributions to memcp. If you would like to contribute, please follow these steps:

- Fork the repository and create a new branch for your changes.
- Make your changes and commit them to your branch.
- Push your branch to your fork and create a pull request.

Before submitting a pull request, please make sure that your changes pass the existing tests and add new tests if necessary.

# Further Reading

- https://launix.de/launix/how-to-balance-a-database-between-olap-and-oltp-workflows/
- https://launix.de/launix/designing-a-programming-language-for-distributed-systems-and-highly-parallel-algorithms/
- https://launix.de/launix/on-designing-an-interface-for-columnar-in-memory-storage-in-golang/
- https://launix.de/launix/how-in-memory-compression-affects-performance/
- https://launix.de/launix/memory-efficient-indices-for-in-memory-storages/
