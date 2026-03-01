all: jitgen
	go build

jitgen:
	go run ./tools/jitgen/ -patch scm/alu.go scm/list.go scm/strings.go scm/scm.go scm/date.go scm/streams.go scm/sync.go scm/metrics.go scm/scheduler.go scm/window.go scm/vector.go scm/packrat.go scm/jit.go

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

.PHONY: memcp.sif docs jitgen
