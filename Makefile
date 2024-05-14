all:
	go build

run:
	./memcp

perf:
	perf record --call-graph fp -- ./memcp

memcp.sif:
	sudo singularity build memcp.sif memcp.singularity.recipe

.PHONY: memcp.sif
