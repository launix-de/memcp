# Arithmetic / Logic

## int?

tells if the value is a integer

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): value

### Returns

`bool`

## number?

tells if the value is a number

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): value

### Returns

`bool`

## +

adds two or more numbers

**Allowed number of parameters:** 2–1000

### Parameters

- **value...** (`number`): values to add

### Returns

`number`

## -

subtracts two or more numbers from the first one

**Allowed number of parameters:** 2–1000

### Parameters

- **value...** (`number`): values

### Returns

`number`

## *

multiplies two or more numbers

**Allowed number of parameters:** 2–1000

### Parameters

- **value...** (`number`): values

### Returns

`number`

## /

divides two or more numbers from the first one

**Allowed number of parameters:** 2–1000

### Parameters

- **value...** (`number`): values

### Returns

`number`

## <=

compares two numbers or strings

**Allowed number of parameters:** 2–2

### Parameters

- **value...** (`any`): values

### Returns

`bool`

## <

compares two numbers or strings

**Allowed number of parameters:** 2–2

### Parameters

- **value...** (`any`): values

### Returns

`bool`

## >

compares two numbers or strings

**Allowed number of parameters:** 2–2

### Parameters

- **value...** (`any`): values

### Returns

`bool`

## >=

compares two numbers or strings

**Allowed number of parameters:** 2–2

### Parameters

- **value...** (`any`): values

### Returns

`bool`

## equal?

compares two values of the same type, (equal? nil nil) is true

**Allowed number of parameters:** 2–2

### Parameters

- **value...** (`any`): values

### Returns

`bool`

## equal??

performs a SQL compliant sloppy equality check on primitive values (number, int, string, bool. nil), strings are compared case insensitive, (equal? nil nil) is nil

**Allowed number of parameters:** 2–2

### Parameters

- **value...** (`any`): values

### Returns

`bool`

## !

negates the boolean value

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`bool`): value

### Returns

`bool`

## not

negates the boolean value

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`bool`): value

### Returns

`bool`

## nil?

returns true if value is nil

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): value

### Returns

`bool`

## min

returns the smallest value

**Allowed number of parameters:** 1–1000

### Parameters

- **value...** (`number|string`): value

### Returns

`number|string`

## max

returns the highest value

**Allowed number of parameters:** 1–1000

### Parameters

- **value...** (`number|string`): value

### Returns

`number|string`

## floor

rounds the number down

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`number`): value

### Returns

`number`

## ceil

rounds the number up

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`number`): value

### Returns

`number`

## round

rounds the number

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`number`): value

### Returns

`number`

