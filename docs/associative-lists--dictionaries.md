# Associative Lists / Dictionaries

## filter_assoc

returns a filtered dictionary according to a filter function

**Allowed number of parameters:** 2–2

### Parameters

- **dict** (`list`): dictionary that has to be filtered
- **condition** (`func`): filter function func(string any)->bool where the first parameter is the key, the second is the value

### Returns

`list`

## map_assoc

returns a mapped dictionary according to a map function
Keys will stay the same but values are mapped.

**Allowed number of parameters:** 2–2

### Parameters

- **dict** (`list`): dictionary that has to be mapped
- **map** (`func`): map function func(string any)->any where the first parameter is the key, the second is the value. It must return the new value.

### Returns

`list`

## reduce_assoc

reduces a dictionary according to a reduce function

**Allowed number of parameters:** 3–3

### Parameters

- **dict** (`list`): dictionary that has to be reduced
- **reduce** (`func`): reduce function func(any string any)->any where the first parameter is the accumulator, second is key, third is value. It must return the new accumulator.
- **neutral** (`any`): initial value for the accumulator

### Returns

`any`

## has_assoc?

checks if a dictionary has a key present

**Allowed number of parameters:** 2–2

### Parameters

- **dict** (`list`): dictionary that has to be checked
- **key** (`string`): key to test

### Returns

`bool`

## extract_assoc

applies a function (key value) on the dictionary and returns the results as a flat list

**Allowed number of parameters:** 2–2

### Parameters

- **dict** (`list`): dictionary that has to be checked
- **map** (`func`): func(string any)->any that flattens down each element

### Returns

`list`

## set_assoc

returns a dictionary where a single value has been changed.
This function may destroy the input value for the sake of performance. You must not use the input value again.

**Allowed number of parameters:** 3–4

### Parameters

- **dict** (`list`): input dictionary that has to be changed. You must not use this value again.
- **key** (`string`): key that has to be set
- **value** (`any`): new value to set
- **merge** (`func`): (optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value. It must return the merged value that shall be pysically stored in the new dictionary.

### Returns

`list`

## merge_assoc

returns a dictionary where all keys from dict1 and all keys from dict2 are present.
If a key is present in both inputs, the second one will be dominant so the first value will be overwritten unless you provide a merge function

**Allowed number of parameters:** 2–3

### Parameters

- **dict1** (`list`): first input dictionary that has to be changed. You must not use this value again.
- **dict2** (`list`): input dictionary that contains the new values that have to be added
- **merge** (`func`): (optional) func(any any)->any that is called when a value is overwritten. The first parameter is the old value, the second is the new value from dict2. It must return the merged value that shall be pysically stored in the new dictionary.

### Returns

`list`

