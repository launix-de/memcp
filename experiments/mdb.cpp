#include <lmdb.h>
#include <cstdio>
// apt install liblmbd-dev

int main2() {
	MDB_env *db;
	mdb_env_create(&db); // init
	mdb_env_open(db, "test.db/", 0, 0755); // TODO: read db from parameter
	// TODO: mdb_env_set_mapsize max size setzen

	// test txn
	MDB_txn *txn;
	mdb_txn_begin(db, 0, 0, &txn); // MDB_RDONLY
	MDB_dbi dbi;
	auto err = mdb_dbi_open(txn, 0, MDB_CREATE, &dbi);
	if (err) {
		printf("err = %d\n", err); // todo exit
	} else {
		printf("dbi = %d\n", dbi);
	}
	
	// read counter and increase
	MDB_val count;
	int newcount;
	MDB_val key_count {5, (void*) "count"};
	if (mdb_get(txn, dbi, &key_count, &count)) {
		// not found
		printf("Opened the first time\n");
		newcount = 0;
		count.mv_size = sizeof(newcount);
		count.mv_data = &newcount;
	} else {
		newcount = *((int*) count.mv_data);
		count.mv_size = sizeof(newcount);
		count.mv_data = &newcount;
	}
	newcount++;
	printf("Counter = %d\n", newcount);
	mdb_put(txn, dbi, &key_count, &count, 0);
	// mdb_del gibts auch noch
	mdb_txn_commit(txn); // oder abort, reset, renew (readonly)


	mdb_env_close(db); // close db (no transaction, no cursor must be open)
	return 0;
}


