package main

import "os"
import "bufio"
//import "fmt"
import "encoding/json"

// todo: enhance table datatype
type dataset map[string]scmer
type table []dataset

var tables map[string]table = make(map[string]table)

func loadStorageFrom(filename string) {
	f, _ := os.Open(filename)
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	var t string
	for scanner.Scan() {
		s := scanner.Text()
		if s == "" {
			// ignore
		} else if s[0:7] == "#table " {
			// new table
			t = s[7:]
		} else if s[0] == '#' {
			// comment
		} else {
			var x dataset
			json.Unmarshal([]byte(s), &x) // parse JSON
			tables[t] = append(tables[t], x) // put into table
		}
	}
}

func initStorageEngine(en env) {
	// example: (scan "PLZ" (lambda () 1) (lambda (PLZ Ort) (print PLZ " - " Ort)))
	// example: (scan "PLZ" (lambda (Ort) (equal? Ort "Neugersdorf")) (lambda (PLZ Ort) (print PLZ " - " Ort)))
	// example: (scan "PLZ" (lambda (Ort) (equal? Ort "Dresden")) (lambda (PLZ Ort) (print PLZ " - " Ort)))
	en.vars["scan"] = func (a ...scmer) scmer {
		// params: table, condition, map, reduce, reduceSeed
		t := tables[a[0].(string)]
		cargs := a[1].(proc).params.([]scmer) // list of arguments condition
		margs := a[2].(proc).params.([]scmer) // list of arguments map
		cdataset := make([]scmer, len(cargs))
		mdataset := make([]scmer, len(margs))

		// TODO: analyze condition and find indexes

		for _, item := range t { // iterate over table
			for i, k := range cargs { // iterate over columns
				cdataset[i] = item[string(k.(symbol))] // fill value
			}
			// check condition
			if (toBool(apply(a[1], cdataset))) {
				// call map function
				for i, k := range margs { // iterate over columns
					mdataset[i] = item[string(k.(symbol))] // fill value
				}
				apply(a[2], mdataset) // todo: output monad
			}
		}
		return "ok"
	}
}
