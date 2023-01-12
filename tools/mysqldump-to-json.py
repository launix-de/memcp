#!/bin/env python

import mysql.connector
import simplejson as json
import argparse

parser = argparse.ArgumentParser()
parser.add_argument('-H', '--host', default='localhost', help='hostname')
parser.add_argument('-u', '--user', required=True, help='user')
parser.add_argument('-p', '--password', required=True, help='password')
parser.add_argument('database', help='database')
args = parser.parse_args()

hostname = args.host
user = args.user
password = args.password
database = args.database or user


mydb = mysql.connector.connect(
  host=hostname,
  user=user,
  password=password,
  database=database
)

mycursor = mydb.cursor()
mycursor.execute("SHOW TABLES")

tables = []
for x in mycursor:
	tables.append(x[0])

for t in tables:
	print('#table ' + t)
	mycursor.execute("SELECT * FROM `"+t.replace("`", "``")+"`")
	print('#columns ', mycursor.column_names)
	for row in mycursor:
		print(json.dumps(dict(zip(mycursor.column_names, row))))
	print('')
