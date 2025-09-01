# Streams

## streamString

creates a stream that contains a string

**Allowed number of parameters:** 1–1

### Parameters

- **content** (`string`): content to put into the stream

### Returns

`stream`

## gzip

compresses a stream with gzip. Create streams with (stream filename)

**Allowed number of parameters:** 1–1

### Parameters

- **stream** (`stream`): input stream

### Returns

`stream`

## xz

compresses a stream with xz. Create streams with (stream filename)

**Allowed number of parameters:** 1–1

### Parameters

- **stream** (`stream`): input stream

### Returns

`stream`

## zcat

turns a compressed gzip stream into a stream of uncompressed data. Create streams with (stream filename)

**Allowed number of parameters:** 1–1

### Parameters

- **stream** (`stream`): input stream

### Returns

`stream`

## xzcat

turns a compressed xz stream into a stream of uncompressed data. Create streams with (stream filename)

**Allowed number of parameters:** 1–1

### Parameters

- **stream** (`stream`): input stream

### Returns

`stream`

