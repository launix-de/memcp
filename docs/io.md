# IO

## print

Prints values to stdout (only in IO environment)

**Allowed number of parameters:** 1–1000

### Parameters

- **value...** (`any`): values to print

### Returns

`bool`

## env

returns the content of a environment variable

**Allowed number of parameters:** 1–2

### Parameters

- **var** (`string`): envvar
- **default** (`string`): default if the env is not found

### Returns

`string`

## help

Lists all functions or print help for a specific function

**Allowed number of parameters:** 0–1

### Parameters

- **topic** (`string`): function to print help about

### Returns

`nil`

## import

Imports a file .scm file into current namespace

**Allowed number of parameters:** 1–1

### Parameters

- **filename** (`string`): filename relative to folder of source file

### Returns

`any`

## load

Loads a file or stream and returns the string or iterates line-wise

**Allowed number of parameters:** 1–3

### Parameters

- **filenameOrStream** (`string|stream`): filename relative to folder of source file or stream to read from
- **linehandler** (`func`): handler that reads each line; each line may end with delimiter
- **delimiter** (`string`): delimiter to extract; if no delimiter is given, the file is read as whole and returned or passed to linehandler

### Returns

`string|bool`

## stream

Opens a file readonly as stream

**Allowed number of parameters:** 1–1

### Parameters

- **filename** (`string`): filename relative to folder of source file

### Returns

`stream`

## watch

Loads a file and calls the callback. Whenever the file changes on disk, the file is load again.

**Allowed number of parameters:** 2–2

### Parameters

- **filename** (`string`): filename relative to folder of source file
- **updatehandler** (`func`): handler that receives the file content func(content)

### Returns

`bool`

## serve

Opens a HTTP server at a given port

**Allowed number of parameters:** 2–2

### Parameters

- **port** (`number`): port number for HTTP server
- **handler** (`func`): handler: lambda(req res) that handles the http request (TODO: detailed documentation)

### Returns

`bool`

## serveStatic

creates a static handler for use as a callback in (serve) - returns a handler lambda(req res)

**Allowed number of parameters:** 1–1

### Parameters

- **directory** (`string`): folder with the files to serve

### Returns

`func`

## mysql

Imports a file .scm file into current namespace

**Allowed number of parameters:** 4–4

### Parameters

- **port** (`number`): port number for MySQL server
- **getPassword** (`func`): lambda(username string) string|nil has to return the password for a user or nil to deny login
- **schemacallback** (`func`): lambda(username schema) bool handler check whether user is allowed to schem (string) - you should check access rights here
- **handler** (`func`): lambda(schema sql resultrow session) handler to process sql query (string) in schema (string). resultrow is a lambda(list)

### Returns

`bool`

## password

Hashes a password with sha1 (for mysql user authentication)

**Allowed number of parameters:** 1–1

### Parameters

- **password** (`string`): plain text password to hash

### Returns

`string`

## args

Returns command line arguments

**Allowed number of parameters:** 0–0

### Parameters

_This function has no parameters._

### Returns

`list`

## arg

Gets a command line argument value

**Allowed number of parameters:** 2–3

### Parameters

- **longname** (`string`): long argument name (without --)
- **shortname** (`string`): short argument name (without -) or default value if only 2 args
- **default** (`any`): default value if argument not found

### Returns

`any`

