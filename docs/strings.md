# Strings

## string?

tells if the value is a string

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): value

### Returns

`bool`

## concat

concatenates stringable values and returns a string

**Allowed number of parameters:** 1–1000

### Parameters

- **value...** (`any`): values to concat

### Returns

`string`

## substr

returns a substring

**Allowed number of parameters:** 2–3

### Parameters

- **value** (`string`): string to cut
- **start** (`number`): first character index
- **len** (`number`): optional length

### Returns

`string`

## simplify

turns a stringable input value in the easiest-most value (e.g. turn strings into numbers if they are numeric

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): value to simplify

### Returns

`any`

## strlen

returns the length of a string

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): input string

### Returns

`int`

## strlike

matches the string against a wildcard pattern (SQL compliant)

**Allowed number of parameters:** 2–3

### Parameters

- **value** (`string`): input string
- **pattern** (`string`): pattern with % and _ in them
- **collation** (`string`): collation in which to compare them

### Returns

`bool`

## toLower

turns a string into lower case

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): input string

### Returns

`string`

## toUpper

turns a string into upper case

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): input string

### Returns

`string`

## replace

replaces all occurances in a string with another string

**Allowed number of parameters:** 3–3

### Parameters

- **s** (`string`): input string
- **find** (`string`): search string
- **replace** (`string`): replace string

### Returns

`string`

## split

splits a string using a separator or space

**Allowed number of parameters:** 1–2

### Parameters

- **value** (`string`): input string
- **separator** (`string`): (optional) parameter, defaults to " "

### Returns

`list`

## collate

returns the `<` operator for a given collation. MemCP allows natural sorting of numeric literals.

**Allowed number of parameters:** 1–1

### Parameters

- **collation** (`string`): collation string of the form LANG or LANG_cs or LANG_ci where LANG is a BCP 47 code, for compatibility to MySQL, a CHARSET_ prefix is allowed and ignored as well as the aliases bin, danish, general, german1, german2, spanish and swedish are allowed for language codes
- **reverse** (`bool`): whether to reverse the order like in ORDER BY DESC

### Returns

`func`

## htmlentities

escapes the string for use in HTML

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): input string

### Returns

`string`

## urlencode

encodes a string according to URI coding schema

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): string to encode

### Returns

`string`

## urldecode

decodes a string according to URI coding schema

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): string to decode

### Returns

`string`

## json_encode

encodes a value in JSON, treats lists as lists

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): value to encode

### Returns

`string`

## json_encode_assoc

encodes a value in JSON, treats lists as associative arrays

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): value to encode

### Returns

`string`

## json_decode

parses JSON into a map

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): string to decode

### Returns

`any`

## sql_unescape

unescapes the inner part of a sql string

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): string to decode

### Returns

`string`

## bin2hex

turns binary data into hex with lowercase letters

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): string to decode

### Returns

`string`

## hex2bin

decodes a hex string into binary data

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`string`): hex string (even length)

### Returns

`string`

## randomBytes

returns a string with numBytes cryptographically secure random bytes

**Allowed number of parameters:** 1–1

### Parameters

- **numBytes** (`number`): number of random bytes

### Returns

`string`

