all:
	go build

run:
	./memcp

perf:
	perf record --call-graph fp -- ./memcp
