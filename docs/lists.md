# Lists

## count

counts the number of elements in the list

**Allowed number of parameters:** 1–1

### Parameters

- **list** (`list`): base list

### Returns

`int`

## nth

get the nth item of a list

**Allowed number of parameters:** 2–2

### Parameters

- **list** (`list`): base list
- **index** (`number`): index beginning from 0

### Returns

`any`

## append

appends items to a list and return the extended list.
The original list stays unharmed.

**Allowed number of parameters:** 2–1000

### Parameters

- **list** (`list`): base list
- **item...** (`any`): items to add

### Returns

`list`

## append_unique

appends items to a list but only if they are new.
The original list stays unharmed.

**Allowed number of parameters:** 2–1000

### Parameters

- **list** (`list`): base list
- **item...** (`any`): items to add

### Returns

`list`

## cons

constructs a list from a head and a tail list

**Allowed number of parameters:** 2–2

### Parameters

- **car** (`any`): new head element
- **cdr** (`list`): tail that is appended after car

### Returns

`list`

## car

extracts the head of a list

**Allowed number of parameters:** 1–1

### Parameters

- **list** (`list`): list

### Returns

`any`

## cdr

extracts the tail of a list
The tail of a list is a list with all items except the head.

**Allowed number of parameters:** 1–1

### Parameters

- **list** (`list`): list

### Returns

`any`

## zip

swaps the dimension of a list of lists. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as the components that will be zipped into the sub list

**Allowed number of parameters:** 1–1000

### Parameters

- **list** (`list`): list of lists of items

### Returns

`list`

## merge

flattens a list of lists into a list containing all the subitems. If one parameter is given, it is a list of lists that is flattened. If multiple parameters are given, they are treated as lists that will be merged into one

**Allowed number of parameters:** 1–1000

### Parameters

- **list** (`list`): list of lists of items

### Returns

`list`

## merge_unique

flattens a list of lists into a list containing all the subitems. Duplicates are filtered out.

**Allowed number of parameters:** 1–1000

### Parameters

- **list** (`list`): list of lists of items

### Returns

`list`

## has?

checks if a list has a certain item (equal?)

**Allowed number of parameters:** 2–2

### Parameters

- **haystack** (`list`): list to search in
- **needle** (`any`): item to search for

### Returns

`bool`

## filter

returns a list that only contains elements that pass the filter function

**Allowed number of parameters:** 2–2

### Parameters

- **list** (`list`): list that has to be filtered
- **condition** (`func`): filter condition func(any)->bool

### Returns

`list`

## map

returns a list that contains the results of a map function that is applied to the list

**Allowed number of parameters:** 2–2

### Parameters

- **list** (`list`): list that has to be mapped
- **map** (`func`): map function func(any)->any that is applied to each item

### Returns

`list`

## mapIndex

returns a list that contains the results of a map function that is applied to the list

**Allowed number of parameters:** 2–2

### Parameters

- **list** (`list`): list that has to be mapped
- **map** (`func`): map function func(i, any)->any that is applied to each item

### Returns

`list`

## reduce

returns a list that contains the result of a map function

**Allowed number of parameters:** 2–3

### Parameters

- **list** (`list`): list that has to be reduced
- **reduce** (`func`): reduce function func(any any)->any where the first parameter is the accumulator, the second is a list item
- **neutral** (`any`): (optional) initial value of the accumulator, defaults to nil

### Returns

`any`

## produce

returns a list that contains produced items - it works like for(state = startstate, condition(state), state = iterator(state)) {yield state}

**Allowed number of parameters:** 3–3

### Parameters

- **startstate** (`any`): start state to begin with
- **condition** (`func`): func that returns true whether the state will be inserted into the result or the loop is stopped
- **iterator** (`func`): func that produces the next state

### Returns

`list`

## produceN

returns a list with numbers from 0..n-1

**Allowed number of parameters:** 1–1

### Parameters

- **n** (`number`): number of elements to produce

### Returns

`list`

## list?

checks if a value is a list

**Allowed number of parameters:** 1–1

### Parameters

- **value** (`any`): value to check

### Returns

`bool`

## contains?

checks if a value is in a list; uses the equal?? operator

**Allowed number of parameters:** 2–2

### Parameters

- **list** (`list`): list to check
- **value** (`any`): value to check

### Returns

`bool`

