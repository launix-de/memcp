all:
	go build

run:
	./memcp

perf:
	perf record --call-graph fp -- ./memcp

test:
	# run `cp git-pre-commit .git/hooks/pre-commit` to activate the trigger
	./git-pre-commit

memcp.sif:
	sudo singularity build memcp.sif memcp.singularity.recipe

.PHONY: memcp.sif
