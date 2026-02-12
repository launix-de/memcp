all:
	go build

ceph:
	go build -tags=ceph

run:
	./memcp

perf:
	perf record --call-graph fp -- ./memcp

test:
	# run `cp git-pre-commit .git/hooks/pre-commit` to activate the trigger
	MEMCP_COVERAGE=1 MEMCP_COVERDIR=/tmp/memcp-coverage ./git-pre-commit

memcp.sif:
	sudo singularity build memcp.sif memcp.singularity.recipe

docs:
	./memcp -write-docu docs

docker-release:
	sudo docker build -t carli2/memcp:latest .
	sudo docker push carli2/memcp:latest

.PHONY: memcp.sif docs
