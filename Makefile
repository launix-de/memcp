all:
	go build
	# Build PHP plugin (stub) as a Go plugin
	go build -buildmode=plugin -o php/memcp_php.so ./php

run:
	./memcp

perf:
	perf record --call-graph fp -- ./memcp

test:
	# run `cp git-pre-commit .git/hooks/pre-commit` to activate the trigger
	./git-pre-commit

memcp.sif:
	sudo singularity build memcp.sif memcp.singularity.recipe

docs:
	./memcp -write-docu docs

docker-release:
	sudo docker build -t carli2/memcp:latest .
	sudo docker push carli2/memcp:latest

.PHONY: memcp.sif docs
