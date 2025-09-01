# Sync

## newsession

Creates a new session which is a threadsafe key-value store represented as a function that can be either called as a getter (session key) or setter (session key value) or list all keys with (session)

**Allowed number of parameters:** 0–0

### Parameters

_This function has no parameters._

### Returns

`func`

## context

Context helper function. Each context also contains a session. (context func args) creates a new context and runs func in that context, (context "session") reads the session variable, (context "check") will check the liveliness of the context and otherwise throw an error

**Allowed number of parameters:** 1–1000

### Parameters

- **args...** (`any`): depends on the usage

### Returns

`any`

## sleep

sleeps the amount of seconds

**Allowed number of parameters:** 1–1

### Parameters

- **duration** (`number`): number of seconds to sleep

### Returns

`bool`

## once

Creates a function wrapper that you can call multiple times but only gets executed once. The result value is cached and returned on a second call. You can add parameters to that resulting function that will be passed to the first run of the wrapped function.

**Allowed number of parameters:** 1–1

### Parameters

- **f** (`func`): function that produces the result value

### Returns

`func`

## mutex

Creates a mutex. The return value is a function that takes one parameter which is a parameterless function. The mutex is guaranteed that all calls to that mutex get serialized.

**Allowed number of parameters:** 1–1

### Parameters

_This function has no parameters._

### Returns

`func`

