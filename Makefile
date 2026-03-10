PREFIX      ?= /usr/local
SYSTEMD_DIR ?= $(PREFIX)/lib/systemd/system

all:
	go build

ceph:
	go build -tags=ceph

install: all
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 memcp $(DESTDIR)$(PREFIX)/bin/memcp
	install -d $(DESTDIR)$(PREFIX)/lib/memcp/lib
	install -m 644 lib/*.scm $(DESTDIR)$(PREFIX)/lib/memcp/lib/
	install -d $(DESTDIR)$(PREFIX)/lib/memcp/assets
	install -m 644 assets/* $(DESTDIR)$(PREFIX)/lib/memcp/assets/
	install -d $(DESTDIR)$(SYSTEMD_DIR)
	install -m 644 memcp.service $(DESTDIR)$(SYSTEMD_DIR)/memcp.service
	@if [ -n "$(DESTDIR)" ]; then \
		install -d $(DESTDIR)/etc/memcp; \
		install -m 640 debian/memcp.conf.default $(DESTDIR)/etc/memcp/memcp.conf; \
		chgrp memcp $(DESTDIR)/etc/memcp/memcp.conf 2>/dev/null || true; \
	else \
		[ -f /etc/memcp/memcp.conf ] || (install -d /etc/memcp && install -m 640 debian/memcp.conf.default /etc/memcp/memcp.conf && chgrp memcp /etc/memcp/memcp.conf 2>/dev/null || true); \
		systemctl daemon-reload; \
		systemctl enable memcp; \
		if systemctl is-active --quiet memcp; then \
			systemctl restart memcp; \
		else \
			systemctl start memcp; \
		fi \
	fi

run:
	./memcp

perf:
	perf record --call-graph fp -- ./memcp

test:
	# run `cp git-pre-commit .git/hooks/pre-commit` to activate the trigger
	MEMCP_COVERAGE=1 MEMCP_COVERDIR=/tmp/memcp-coverage ./git-pre-commit

memcp.sif:
	sudo singularity build memcp.sif memcp.singularity.recipe

# Version is the first word of the first line of CHANGELOG.md (e.g. "0.2")
VERSION     ?= $(shell head -1 CHANGELOG.md | awk '{print $$1}')

DEB_ARCH    ?= $(shell dpkg --print-architecture 2>/dev/null || echo amd64)
DEB_DIR     := memcp_$(VERSION)_$(DEB_ARCH)
DEB_OUT     := memcp_$(VERSION)_$(DEB_ARCH).deb

# RPM uses the native arch name (x86_64, aarch64, …)
RPM_ARCH    ?= $(shell uname -m)
RPM_OUT     := memcp_$(VERSION)_$(RPM_ARCH).rpm

memcp.deb: $(DEB_OUT)
$(DEB_OUT): all
	rm -rf $(DEB_DIR)
	mkdir -p $(DEB_DIR)/DEBIAN
	$(MAKE) install DESTDIR=$(DEB_DIR) PREFIX=/usr SYSTEMD_DIR=/usr/lib/systemd/system
	printf "Package: memcp\nVersion: $(VERSION)\nArchitecture: $(DEB_ARCH)\nMaintainer: Carl-Philip Hänsch <hänsch@launix.de>\nDescription: memcp smart clusterable distributed database\n" \
		> $(DEB_DIR)/DEBIAN/control
	install -m 755 debian/postinst $(DEB_DIR)/DEBIAN/postinst
	install -m 755 debian/prerm    $(DEB_DIR)/DEBIAN/prerm
	install -m 755 debian/postrm   $(DEB_DIR)/DEBIAN/postrm
	echo "/etc/memcp/memcp.conf" > $(DEB_DIR)/DEBIAN/conffiles
	dpkg-deb --build --root-owner-group $(DEB_DIR) $(DEB_OUT)
	rm -rf $(DEB_DIR)

memcp.rpm: $(RPM_OUT)
$(RPM_OUT): all
	mkdir -p .rpmbuild/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
	rpmbuild -bb memcp.spec \
		--define "_topdir $(PWD)/.rpmbuild" \
		--define "_version $(VERSION)" \
		--define "_arch $(RPM_ARCH)" \
		--define "_srcdir $(PWD)"
	find .rpmbuild/RPMS/$(RPM_ARCH)/ -name 'memcp-*.rpm' -exec cp {} $(RPM_OUT) \;
	rm -rf .rpmbuild

docs:
	./memcp -write-docu docs

docker-release:
	sudo docker build -t carli2/memcp:$(VERSION) -t carli2/memcp:latest .
	sudo docker push carli2/memcp:$(VERSION)
	sudo docker push carli2/memcp:latest

.PHONY: memcp.sif memcp.deb memcp.rpm docs docker-release
