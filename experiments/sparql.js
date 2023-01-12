var kdb = [
	['p1', 'name', 'peter'],
	['p1', 'beruf', 'programmierer'],
	['p2', 'name', 'lisa'],
	['p2', 'beruf', 'programmierer'],
	['p3', 'name', 'claudia']
];


/* Syntax von SPARQL:
 SELECT ?var, ?var WHERE {
	 a b c .
	 a b c .
 }
 mit "string"
 mit ?var
 mit prefix:string
 mit prefix:"string"

 JSON representation:
 {
	 PREFIX: {
		 'lx': 'https://launix.de/ontology/person.json#'
	 },
	 SELECT: ['var', 'var', 'var'],
	 WHERE: [
	 	['a', 'b', 'c'],
		['var', {subquery}]
	 ]
 }
 wobei strings folgendes sein können:
 'asdf' literal
 ['prefix', 'string'] geprefixt
 ['var'] variable
 ['prefix', ['var']] pattern
 [] blank

 subqueries werden zum Schluss ausgewertet und ergeben ein array[object] so wie die hauptquery

 Aufbau der Storage Engine:
 SPO-Index als Prefix Tree mit einem Dictionary Compressed String format ?!
 Characters 0-255 sind ASCII, alles darüber sind Tupel

 Prefix dictionary!
  - man hat ein Grundalphabet, z.B. {0,1} oder [0..255] -> Literale
  - Das Alphabet wird erweitert als Tupelbaum -> Prefix+Literal
  - jedes Wort kann man als einen Buchstaben darstellen
  - btree mit folgender Struktur (64B/line)
    * [ptr64] parent
    * [byte] size of prefix
    * n x [byte] prefix
    * [byte] number of subnodes
    * n x [byte] next-characters
    * padding for 8-byte alignment
    * n x [ptr64] next-node
    * [ptr64] payload ptr

    - Paul, Anna, Peter ->
    	n0: 0, 2, A, P, x, x, x, x, [n1], [n2]
	n1: 4, A, n, n, a, 0
	n2: 1, P, 2, e, t, x, x, x, [n3], [n4]
	n3: 3, a, u, l, 0
	n4: 4, e, t, e, r, 0

*/

var q = {
	SELECT: ['name'],
	WHERE: [
		[['p'], 'name', ['name']],
		[['p'], 'beruf', 'programmierer']
	]
}

function* query(q, kdb) {
	// backtracing algo
	var vars = {};
	function unify(a, b, vars) {
		// pre: b does not contain patterns! (b is from a knowledge base)
		if (typeof a === 'string') {
			// literal
			if (typeof b === 'string') {
				return a === b;
			} else if (b.constructor === Array) {
				if (b.length === 1) {
					throw 'this should not be possible (pattern on rhs)';
					/*
					if (vars[b[0]] === undefined) {
						vars[b[0]] = a; // assign var
						return true;
					} else {
						return vars[b[0]] === a; // compare var
					}*/
				} else {
					throw 'not implemented (TODO: pfx)';
				}
			}
		} else if (a.constructor === Array) {
			// var oder pfx
			if (a.length === 1) {
				if (typeof b === 'string') {
					if (vars[a[0]] === undefined) {
						vars[a[0]] = b; // assign var
						return true;
					} else {
						return vars[a[0]] === b; // compare var
					}
				} else if (b.constructor === Array) {
					if (b.length === 1) {
						// var <-> var
						throw 'this should not be possible (pattern on rhs)';
					} else {
						throw 'not implemented (TODO: pfx)';
					}
				}
			} else {
				throw 'not implemented (TODO: pfx)';
			}
		}
		throw 'unhandled case';
	}
	function* match(i, vars) {
		if (i < q.WHERE.length) {
			for (var j = 0; j < kdb.length; j++) {
				var vars2 = Object.create(vars);
				if (unify(q.WHERE[i][0], kdb[j][0], vars2) && unify(q.WHERE[i][1], kdb[j][1], vars2) && unify(q.WHERE[i][2], kdb[j][2], vars2)) {
					for (var x of match(i + 1, vars2)) {
						yield x;
					}
				}
			}
		} else {
			var result = {};
			for (prop of q.SELECT) {
				result[prop] = vars[prop];
			}
			yield result; // ein Result
		}
	}
	for (var x of match(0, vars)) {
		yield x;
	}
}

for (var x of query(q, kdb)) {
	console.log(x);
}
