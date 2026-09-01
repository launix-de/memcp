# Copyright (C) 2023 - 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

PREFIX       ?= /usr/local
SYSTEMD_DIR  ?= $(PREFIX)/lib/systemd/system
GOOS         ?= linux
GOARCH       ?= $(shell go env GOARCH)
CGO_ENABLED  ?= 0
BUILD_FLAGS  ?= -trimpath -buildvcs=false
LDFLAGS      ?=
PACKAGE_LDFLAGS ?= -s -w
DIST_DIR     ?= dist
PACKAGE_DIR  ?= .build/packages
SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct)
export SOURCE_DATE_EPOCH

all:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) -ldflags="$(LDFLAGS)" -o memcp .

jitgen:
	@set -eu; \
	jitgen_bin=$$(mktemp /tmp/memcp-jitgen.XXXXXX); \
	trap 'rm -f "$$jitgen_bin"' EXIT; \
	go build -o "$$jitgen_bin" ./tools/jitgen/; \
	"$$jitgen_bin" -patch scm/alu.go scm/list.go scm/strings.go scm/scm.go scm/date.go scm/streams.go scm/sync.go scm/metrics.go scm/scheduler.go scm/window.go scm/vector.go scm/packrat.go scm/jit.go scm/timezone.go scm/processlist.go scm/list_assoc_extra.go scm/sql_literals.go scm/json_functions.go; \
	"$$jitgen_bin" -patch storage/storage-int.go storage/storage-float.go storage/storage-decimal.go storage/storage-string.go storage/storage-prefix.go storage/storage-enum.go storage/storage-scmer.go storage/storage-sparse.go storage/storage-seq.go storage/storage-const.go storage/overlay-blob.go storage/compute_proxy.go; \
	gofmt -w scm storage

jitgen-policy:
	@set -eu; \
	jitgen_bin=$$(mktemp /tmp/memcp-jitgen.XXXXXX); \
	trap 'rm -f "$$jitgen_bin"' EXIT; \
	go build -o "$$jitgen_bin" ./tools/jitgen/; \
	"$$jitgen_bin" -patch -policy-only scm/alu.go scm/list.go scm/strings.go scm/scm.go scm/date.go scm/streams.go scm/sync.go scm/metrics.go scm/scheduler.go scm/window.go scm/vector.go scm/packrat.go scm/jit.go scm/timezone.go scm/processlist.go scm/list_assoc_extra.go scm/sql_literals.go scm/json_functions.go; \
	gofmt -w scm

costgen:
	go run ./tools/costgen -patch

ceph:
	go build -tags=ceph

install: all install-files

install-files:
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 memcp $(DESTDIR)$(PREFIX)/bin/memcp
	install -d $(DESTDIR)$(PREFIX)/lib/memcp/lib
	install -m 644 lib/*.scm $(DESTDIR)$(PREFIX)/lib/memcp/lib/
	install -d $(DESTDIR)$(PREFIX)/lib/memcp/assets
	install -m 644 assets/* $(DESTDIR)$(PREFIX)/lib/memcp/assets/
	install -d $(DESTDIR)$(SYSTEMD_DIR)
	install -m 644 memcp.service $(DESTDIR)$(SYSTEMD_DIR)/memcp.service
	install -d $(DESTDIR)$(PREFIX)/lib/memcp
	install -m 755 packaging/initialize.sh $(DESTDIR)$(PREFIX)/lib/memcp/initialize
	install -d $(DESTDIR)$(PREFIX)/share/doc/memcp
	install -m 644 debian/copyright $(DESTDIR)$(PREFIX)/share/doc/memcp/copyright
	install -m 644 README.md CHANGELOG.md $(DESTDIR)$(PREFIX)/share/doc/memcp/
	@if [ "$(PACKAGE_FORMAT)" = rpm ]; then \
		install -d $(DESTDIR)$(PREFIX)/share/licenses/memcp; \
		install -m 644 LICENSE $(DESTDIR)$(PREFIX)/share/licenses/memcp/LICENSE; \
	fi
	install -d $(DESTDIR)$(PREFIX)/share/man/man1
	gzip -9nc packaging/memcp.1 > $(DESTDIR)$(PREFIX)/share/man/man1/memcp.1.gz
	chmod 644 $(DESTDIR)$(PREFIX)/share/man/man1/memcp.1.gz
	@if [ -n "$(DESTDIR)" ]; then \
		install -d $(DESTDIR)/etc/memcp; \
		install -m 640 debian/memcp.conf.default $(DESTDIR)/etc/memcp/memcp.conf; \
	else \
		install -d /etc/memcp; \
		[ -f /etc/memcp/memcp.conf ] || install -m 640 debian/memcp.conf.default /etc/memcp/memcp.conf; \
	fi

run:
	./memcp

perf:
	perf record --call-graph fp -- ./memcp

test:
	# run `cp git-pre-commit .git/hooks/pre-commit` to activate the trigger
	MEMCP_FAIL_FAST=0 MEMCP_COVERAGE=1 MEMCP_COVERDIR=/tmp/memcp-coverage MEMCP_TEST_DATA_DIR=$$(mktemp -d /tmp/memcp-make-test-data.XXXXXX) ./git-pre-commit

memcp.sif: all
	@if command -v apptainer >/dev/null 2>&1; then \
		apptainer build memcp.sif memcp.singularity.recipe; \
	else \
		singularity build memcp.sif memcp.singularity.recipe; \
	fi

# Version is the first word of the first line of CHANGELOG.md (e.g. "0.2")
VERSION     ?= $(shell head -1 CHANGELOG.md | awk '{print $$1}')

DEB_ARCH    ?= $(shell dpkg --print-architecture 2>/dev/null || echo amd64)
DEB_GOARCH_amd64 := amd64
DEB_GOARCH_arm64 := arm64
DEB_GOARCH        := $(or $(DEB_GOARCH_$(DEB_ARCH)),$(GOARCH))
DEB_DIR           := $(PACKAGE_DIR)/deb-$(VERSION)-$(DEB_ARCH)
DEB_OUT           := $(DIST_DIR)/memcp_$(VERSION)_$(DEB_ARCH).deb

# RPM uses the native arch name (x86_64, aarch64, …)
RPM_ARCH    ?= $(shell uname -m)
RPMBUILD_FLAGS ?=
RPM_GOARCH_x86_64 := amd64
RPM_GOARCH_aarch64 := arm64
RPM_GOARCH         := $(or $(RPM_GOARCH_$(RPM_ARCH)),$(GOARCH))
RPM_OUT            := $(DIST_DIR)/memcp_$(VERSION)_$(RPM_ARCH).rpm
RPM_SOURCE_OUT     := $(DIST_DIR)/memcp_$(VERSION).src.rpm
SOURCE_TREEISH     ?= HEAD

version:
	@printf '%s\n' '$(VERSION)'

artifact-names:
	@printf '%s\n%s\n%s\n' '$(DEB_OUT)' '$(RPM_OUT)' '$(RPM_SOURCE_OUT)'

memcp.deb: $(DEB_OUT)
$(DEB_OUT):
	$(MAKE) all GOOS=linux GOARCH=$(DEB_GOARCH) CGO_ENABLED=0 LDFLAGS="$(PACKAGE_LDFLAGS)"
	rm -rf -- $(DEB_DIR)
	mkdir -p $(DEB_DIR)/DEBIAN
	$(MAKE) install-files DESTDIR=$(DEB_DIR) PREFIX=/usr SYSTEMD_DIR=/usr/lib/systemd/system PACKAGE_FORMAT=deb
	printf "Package: memcp\nVersion: $(VERSION)\nArchitecture: $(DEB_ARCH)\nSection: database\nPriority: optional\nMaintainer: Carl-Philip Hänsch <hänsch@launix.de>\nDepends: adduser\nHomepage: https://github.com/launix-de/memcp\nDescription: smart clusterable distributed database\n MemCP is a persistent, column-oriented in-memory database with HTTP and\n MySQL-compatible interfaces.\n" \
		> $(DEB_DIR)/DEBIAN/control
	install -m 755 debian/postinst $(DEB_DIR)/DEBIAN/postinst
	install -m 755 debian/prerm    $(DEB_DIR)/DEBIAN/prerm
	install -m 755 debian/postrm   $(DEB_DIR)/DEBIAN/postrm
	install -d $(DEB_DIR)/usr/share/lintian/overrides
	install -m 644 debian/memcp.lintian-overrides \
		$(DEB_DIR)/usr/share/lintian/overrides/memcp
	echo "/etc/memcp/memcp.conf" > $(DEB_DIR)/DEBIAN/conffiles
	@deb_date=$$(git log -1 --format=%aD); \
		printf "memcp ($(VERSION)) unstable; urgency=medium\n\n  * Upstream release $(VERSION).\n\n -- Carl-Philip Haensch <haensch@launix.de>  %s\n" "$$deb_date" \
		| gzip -9n > $(DEB_DIR)/usr/share/doc/memcp/changelog.gz
	chmod 644 $(DEB_DIR)/usr/share/doc/memcp/changelog.gz
	@(cd $(DEB_DIR) && find . -type f ! -path './DEBIAN/*' ! -path './etc/memcp/memcp.conf' -print0 \
		| sort -z | xargs -0 md5sum) > $(DEB_DIR)/DEBIAN/md5sums
	mkdir -p $(DIST_DIR)
	dpkg-deb --build --root-owner-group $(DEB_DIR) $(DEB_OUT)
	rm -rf -- $(DEB_DIR)

memcp.rpm: $(RPM_OUT)
$(RPM_OUT):
	rm -rf -- $(PACKAGE_DIR)/rpmbuild
	mkdir -p $(PACKAGE_DIR)/rpmbuild/BUILD $(PACKAGE_DIR)/rpmbuild/RPMS \
		$(PACKAGE_DIR)/rpmbuild/SOURCES $(PACKAGE_DIR)/rpmbuild/SPECS \
		$(PACKAGE_DIR)/rpmbuild/SRPMS $(DIST_DIR)
	git archive --format=tar --prefix=memcp-$(VERSION)/ $(SOURCE_TREEISH) \
		| gzip -9n > $(PACKAGE_DIR)/rpmbuild/SOURCES/memcp-$(VERSION).tar.gz
	rpmbuild $(RPMBUILD_FLAGS) -ba memcp.spec \
		--target "$(RPM_ARCH)" \
		--define "_topdir $(PWD)/$(PACKAGE_DIR)/rpmbuild" \
		--define "_version $(VERSION)" \
		--define "_goarch $(RPM_GOARCH)"
	@rpm_file=$$(find $(PACKAGE_DIR)/rpmbuild/RPMS/$(RPM_ARCH)/ -type f -name 'memcp-*.rpm' -print -quit); \
		test -n "$$rpm_file"; \
		cp "$$rpm_file" $(RPM_OUT)
	@source_rpm=$$(find $(PACKAGE_DIR)/rpmbuild/SRPMS/ -type f -name 'memcp-*.src.rpm' -print -quit); \
		test -n "$$source_rpm"; \
		cp "$$source_rpm" $(RPM_SOURCE_OUT)
	rm -rf -- $(PACKAGE_DIR)/rpmbuild

package-check:
	python3 tools/test_packaging.py --artifacts

docs:
	./memcp -write-docu docs

docker-release:
	docker buildx build --platform linux/amd64,linux/arm64 \
		--provenance=mode=max --sbom=true --push \
		-t carli2/memcp:$(VERSION) -t carli2/memcp:latest .

.PHONY: all install install-files memcp.sif memcp.deb memcp.rpm package-check \
	version artifact-names docs docker-release jitgen costgen
