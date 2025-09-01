# SCM Builtins

## quote

returns a symbol or list without evaluating it

**Allowed number of parameters:** 1–1

### Parameters

- **symbol** (`symbol`): symbol to quote

### Returns

`symbol`

## eval

executes the given scheme program in the current environment

**Allowed number of parameters:** 1–1

### Parameters

- **code** (`list`): list with head and optional parameters

### Returns

`any`

## size

compute the memory size of a value

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): value to examine

### Returns

`int`

## optimize

optimize the given scheme program

**Allowed number of parameters:** 1–1

### Parameters

- **code** (`list`): list with head and optional parameters

### Returns

`any`

## time

measures the time it takes to compute the first argument

**Allowed number of parameters:** 1–2

### Parameters

- **code** (`any`): code to execute
- **label** (`string`): label to print in the log or trace

### Returns

`any`

## if

checks a condition and then conditionally evaluates code branches; there might be multiple condition+true-branch clauses

**Allowed number of parameters:** 2–1000

### Parameters

- **condition...** (`bool`): condition to evaluate
- **true-branch...** (`returntype`): code to evaluate if condition is true
- **false-branch** (`returntype`): code to evaluate if condition is false

### Returns

`returntype`

## and

returns true if all conditions evaluate to true

**Allowed number of parameters:** 1–1000

### Parameters

- **condition** (`bool`): condition to evaluate

### Returns

`bool`

## or

returns true if at least one condition evaluates to true

**Allowed number of parameters:** 1–1000

### Parameters

- **condition** (`any`): condition to evaluate

### Returns

`bool`

## coalesce

returns the first value that has a non-zero value

**Allowed number of parameters:** 1–1000

### Parameters

- **value** (`returntype`): value to examine

### Returns

`returntype`

## coalesceNil

returns the first value that has a non-nil value

**Allowed number of parameters:** 1–1000

### Parameters

- **value** (`returntype`): value to examine

### Returns

`returntype`

## define

defines or sets a variable in the current environment

**Allowed number of parameters:** 2–2

### Parameters

- **variable** (`symbol`): variable to set
- **value** (`returntype`): value to set the variable to

### Returns

`bool`

## set

defines or sets a variable in the current environment

**Allowed number of parameters:** 2–2

### Parameters

- **variable** (`symbol`): variable to set
- **value** (`returntype`): value to set the variable to

### Returns

`bool`

## error

halts the whole execution thread and throws an error message

**Allowed number of parameters:** 1–1000

### Parameters

- **value...** (`any`): value or message to throw

### Returns

`string`

## try

tries to execute a function and returns its result. In case of a failure, the error is fed to the second function and its result value will be used

**Allowed number of parameters:** 2–2

### Parameters

- **func** (`func`): function with no parameters that will be called
- **errorhandler** (`func`): function that takes the error as parameter

### Returns

`any`

## apply

runs the function with its arguments

**Allowed number of parameters:** 2–2

### Parameters

- **function** (`func`): function to execute
- **arguments** (`list`): list of arguments to apply

### Returns

`any`

## apply_assoc

runs the function with its arguments but arguments is a assoc list

**Allowed number of parameters:** 2–2

### Parameters

- **function** (`func`): function to execute (must be a lambda)
- **arguments** (`list`): assoc list of arguments to apply

### Returns

`symbol`

## symbol

returns a symbol built from that string

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): string value that will be converted into a symbol

### Returns

`symbol`

## list

returns a list containing the parameters as alements

**Allowed number of parameters:** 0–10000

### Parameters

- **value...** (`any`): value for the list

### Returns

`list`

## string

converts the given value into string

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): any value

### Returns

`string`

## match

takes a value evaluates the branch that first matches the given pattern
Patterns can be any of:
 - symbol matches any value and stores is into a variable
 - "string" (matches only this string)
 - number (matches only this value)
 - (symbol "something") will only match the symbol 'something'
 - '(subpattern subpattern...) matches a list with exactly these subpatterns
 - (concat str1 str2 str3) will decompose a string into one of the following patterns: "prefix" variable, variable "postfix", variable "infix" variable
 - (cons a b) will reverse the cons function, so it will match the head of the list with a and the rest with b
 - (regex "pattern" text var1 var2...) will match the given regex pattern, store the whole string into text and all capture groups into var1, var2...


**Allowed number of parameters:** 3–10000

### Parameters

- **value** (`any`): value to evaluate
- **pattern...** (`any`): pattern
- **result...** (`returntype`): result value when the pattern matches; this code can use the variables matched in the pattern
- **default** (`any`): (optional) value that is returned when no pattern matches

### Returns

`any`

## lambda

returns a function (func) constructed from the given code

**Allowed number of parameters:** 2–3

### Parameters

- **parameters** (`symbol|list|nil`): if you provide a parameter list, you will have named parameters. If you provide a single symbol, the list of parameters will be provided in that symbol
- **code** (`any`): value that is evaluated when the lambda is called. code can use the parameters provided in the declaration as well es the scope above
- **numvars** (`number`): number of unnamed variables that can be accessed via (var 0) (var 1) etc.

### Returns

`func`

## begin

creates a own variable scope, evaluates all sub expressions and returns the result of the last one

**Allowed number of parameters:** 0–10000

### Parameters

- **expression...** (`any`): expressions to evaluate

### Returns

`any`

## parallel

executes all parameters in parallel and returns nil if they are finished

**Allowed number of parameters:** 1–10000

### Parameters

- **expression...** (`any`): expressions to evaluate in parallel

### Returns

`any`

## source

annotates the node with filename and line information for better backtraces

**Allowed number of parameters:** 4–4

### Parameters

- **filename** (`string`): Filename of the code
- **line** (`number`): Line of the code
- **column** (`number`): Column of the code
- **code** (`returntype`): code

### Returns

`returntype`

## scheme

parses a scheme expression into a list

**Allowed number of parameters:** 1–2

### Parameters

- **code** (`string`): Scheme code
- **filename** (`string`): optional filename

### Returns

`any`

## serialize

serializes a piece of code into a (hopefully) reparsable string; you shall be able to send that code over network and reparse with (scheme)

**Allowed number of parameters:** 1–1

### Parameters

- **code** (`list`): Scheme code

### Returns

`string`

