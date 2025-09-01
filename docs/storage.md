# Storage

## scan

does an unordered parallel filter-map-reduce pass on a single table and returns the reduced result

**Allowed number of parameters:** 6–10

### Parameters

- **schema** (`string|nil`): database where the table is located
- **table** (`string|list`): name of the table to scan (or a list if you have temporary data)
- **filterColumns** (`list`): list of columns that are fed into filter
- **filter** (`func`): lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan
- **mapColumns** (`list`): list of columns that are fed into map
- **map** (`func`): lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '("field1" value1 "field2" value2)) to update certain columns.
- **reduce** (`func`): (optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass.
- **neutral** (`any`): (optional) neutral element for the reduce phase, otherwise nil is assumed
- **reduce2** (`func`): (optional) second stage reduce function that will apply a result of reduce to the neutral element/accumulator
- **isOuter** (`bool`): (optional) if true, in case of no hits, call map once anyway with NULL values

### Returns

`any`

## scan_order

does an ordered parallel filter and serial map-reduce pass on a single table and returns the reduced result

**Allowed number of parameters:** 10–13

### Parameters

- **schema** (`string`): database where the table is located
- **table** (`string`): name of the table to scan
- **filterColumns** (`list`): list of columns that are fed into filter
- **filter** (`func`): lambda function that decides whether a dataset is passed to the map phase. You can use any column of that table as lambda parameter. You should structure your lambda with an (and) at the root element. Every equal? < > <= >= will possibly translated to an indexed scan
- **sortcols** (`list`): list of columns to sort. Each column is either a string to point to an existing column or a func(cols...)->any to compute a sortable value
- **sortdirs** (`list`): list of column directions to sort. Must be same length as sortcols. < means ascending, > means descending, (collate ...) will add collations
- **offset** (`number`): number of items to skip before the first one is fed into map
- **limit** (`number`): max number of items to read
- **mapColumns** (`list`): list of columns that are fed into map
- **map** (`func`): lambda function to extract data from the dataset. You can use any column of that table as lambda parameter. You can return a value you want to extract and pass to reduce, but you can also directly call insert, print or resultrow functions. If you declare a parameter named '$update', this variable will hold a function that you can use to delete or update a row. Call ($update) to delete the dataset, call ($update '("field1" value1 "field2" value2)) to update certain columns.
- **reduce** (`func`): (optional) lambda function to aggregate the map results. It takes two parameters (a b) where a is the accumulator and b the new value. The accumulator for the first reduce call is the neutral element. The return value will be the accumulator input for the next reduce call. There are two reduce phases: shard-local and shard-collect. In the shard-local phase, a starts with neutral and b is fed with the return values of each map call. In the shard-collect phase, a starts with neutral and b is fed with the result of each shard-local pass.
- **neutral** (`any`): (optional) neutral element for the reduce phase, otherwise nil is assumed
- **isOuter** (`bool`): (optional) if true, in case of no hits, call map once anyway with NULL values

### Returns

`any`

## createdatabase

creates a new database

**Allowed number of parameters:** 1–2

### Parameters

- **schema** (`string`): name of the new database
- **ignoreexists** (`bool`): if true, return false instead of throwing an error

### Returns

`bool`

## dropdatabase

drops a database

**Allowed number of parameters:** 1–2

### Parameters

- **schema** (`string`): name of the database
- **ifexists** (`bool`): if true, don't throw an error if it doesn't exist

### Returns

`bool`

## createtable

creates a new database

**Allowed number of parameters:** 4–5

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the new table
- **cols** (`list`): list of columns and constraints, each '("column" colname typename dimensions typeparams) where dimensions is a list of 0-2 numeric items or '("primary" cols) or '("unique" cols) or '("foreign" cols tbl2 cols2 updatemode deletemode of 'restrict'|'cascade'|'set null')
- **options** (`list`): further options like engine=safe|sloppy|memory
- **ifnotexists** (`bool`): don't throw an error if table already exists

### Returns

`bool`

## createcolumn

creates a new column in table

**Allowed number of parameters:** 6–8

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the new table
- **colname** (`string`): name of the new column
- **type** (`string`): name of the basetype
- **dimensions** (`list`): dimensions of the type (e.g. for decimal)
- **options** (`list`): assoc list with one of the following options: primary true, unique true, auto_increment true, null bool, comment string default string collate identifier
- **computorCols** (`list`): list of columns that is passed into params of computor
- **computor** (`func`): lambda expression that can take other column values and computes the value of that column

### Returns

`bool`

## createkey

creates a new key on a table

**Allowed number of parameters:** 5–5

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the new table
- **keyname** (`string`): name of the new key
- **unique** (`bool`): whether the key is unique
- **columns** (`list`): list of columns to include

### Returns

`bool`

## createforeignkey

creates a new foreign key on a table

**Allowed number of parameters:** 8–8

### Parameters

- **schema** (`string`): name of the database
- **keyname** (`string`): name of the new key
- **table1** (`string`): name of the first table
- **columns1** (`list`): list of columns to include
- **table2** (`string`): name of the second table
- **columns2** (`list`): list of columns to include
- **updatemode** (`string`): restrict|cascade|set null
- **deletemode** (`string`): restrict|cascade|set null

### Returns

`bool`

## shardcolumn

tells us how it would partition a column according to their values. Returns a list of pivot elements.

**Allowed number of parameters:** 3–4

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the new table
- **colname** (`string`): name of the column
- **numpartitions** (`number`): number of partitions; optional. leave 0 if you want to detect the partiton number automatically or copy the partition schema of the table

### Returns

`list`

## partitiontable

suggests a partition scheme for a table. If the table has no partition scheme yet, it will immediately apply that scheme and return true. If the table already has a partition scheme, it will alter the partitioning score such that the partitioning scheme is considered in the next repartitioning and return false.

**Allowed number of parameters:** 3–3

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the new table
- **columns** (`list`): associative list of string -> list representing column name -> pivots. You can compute pivots by (shardcolumn ...)

### Returns

`bool`

## altertable

alters a table

**Allowed number of parameters:** 4–4

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the table
- **operation** (`string`): one of owner|drop|engine|collation
- **parameter** (`any`): name of the column to drop or value of the parameter

### Returns

`bool`

## altercolumn

alters a column

**Allowed number of parameters:** 5–5

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the table
- **column** (`string`): name of the column
- **operation** (`string`): one of drop|type|collation|auto_increment|comment
- **parameter** (`any`): name of the column to drop or value of the parameter

### Returns

`bool`

## droptable

removes a table

**Allowed number of parameters:** 2–3

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the table
- **ifexists** (`bool`): if true, don't throw an error if it already exists

### Returns

`bool`

## insert

inserts a new dataset into table and returns the number of successful items

**Allowed number of parameters:** 4–8

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the table
- **columns** (`list`): list of column names, e.g. '("ID", "value")
- **datasets** (`list`): list of list of column values, e.g. '('(1 10) '(2 15))
- **onCollisionCols** (`list`): list of columns of the old dataset that have to be passed to onCollision. Can also request $update.
- **onCollision** (`func`): the function that is called on each collision dataset. The first parameter is filled with the $update function, the second parameter is the dataset as associative list. If not set, an error is thrown in case of a collision.
- **mergeNull** (`bool`): if true, it will handle NULL values as equal according to SQL 2003's definition of DISTINCT (https://en.wikipedia.org/wiki/Null_(SQL)#When_two_nulls_are_equal:_grouping,_sorting,_and_some_set_operations)
- **onInsertid** (`func`): (optional) callback (id)->any; called once with the first auto_increment id assigned for this INSERT

### Returns

`number`

## stat

return memory statistics

**Allowed number of parameters:** 0–2

### Parameters

- **schema** (`string`): name of the database (optional: all databases)
- **table** (`string`): name of the table (if table is set, print the detailled storage stats)

### Returns

`string`

## show

show databases/tables/columns

(show) will list all databases as a list of strings
(show schema) will list all tables as a list of strings
(show schema tbl) will list all columns as a list of dictionaries with the keys (name type dimensions)

**Allowed number of parameters:** 0–2

### Parameters

- **schema** (`string`): (optional) name of the database if you want to list tables or columns
- **table** (`string`): (optional) name of the table if you want to list columns

### Returns

`any`

## rebuild

rebuilds all main storages and returns the amount of time it took

**Allowed number of parameters:** 0–2

### Parameters

- **all** (`bool`): if true, rebuild all shards, even if nothing has changed (default: false)
- **repartition** (`bool`): if true, also repartition (default: true)

### Returns

`string`

## loadCSV

loads a CSV stream into a table and returns the amount of time it took.
The first line of the file must be the headlines. The headlines must match the table's columns exactly.

**Allowed number of parameters:** 3–5

### Parameters

- **schema** (`string`): name of the database
- **table** (`string`): name of the table
- **stream** (`stream`): CSV file, load with: (stream filename)
- **delimiter** (`string`): (optional) delimiter defaults to ";"
- **firstline** (`bool`): (optional) if the first line contains the column names (otherwise, the tables column order is used)

### Returns

`string`

## loadJSON

loads a .jsonl file from stream into a database and returns the amount of time it took.
JSONL is a linebreak separated file of JSON objects. Each JSON object is one dataset in the database. Before you add rows, you must declare the table in a line '#table <tablename>'. All other lines starting with # are comments. Columns are created dynamically as soon as they occur in a json object.

**Allowed number of parameters:** 2–2

### Parameters

- **schema** (`string`): name of the database where you want to put the tables in
- **stream** (`stream`): stream of the .jsonl file, read with: (stream filename)

### Returns

`string`

## settings

reads or writes a global settings value. This modifies your data/settings.json.

**Allowed number of parameters:** 0–2

### Parameters

- **key** (`string`): name of the key to set or get (for reference, rts)
- **value** (`any`): new value of that setting

### Returns

`any`

