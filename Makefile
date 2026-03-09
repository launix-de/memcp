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
		install -m 600 debian/memcp.conf.default $(DESTDIR)/etc/memcp/memcp.conf; \
	else \
		[ -f /etc/memcp/memcp.conf ] || (install -d /etc/memcp && install -m 600 debian/memcp.conf.default /etc/memcp/memcp.conf); \
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

DEB_VERSION ?= $(shell head -1 CHANGELOG.md | tr -d '= \n')
DEB_ARCH    ?= $(shell dpkg --print-architecture 2>/dev/null || echo amd64)
DEB_DIR     := memcp_$(DEB_VERSION)_$(DEB_ARCH)

memcp.deb: all
	rm -rf $(DEB_DIR)
	mkdir -p $(DEB_DIR)/DEBIAN
	$(MAKE) install DESTDIR=$(DEB_DIR) PREFIX=/usr SYSTEMD_DIR=/usr/lib/systemd/system
	printf "Package: memcp\nVersion: $(DEB_VERSION)\nArchitecture: $(DEB_ARCH)\nMaintainer: Carl-Philip Hänsch <hänsch@launix.de>\nDescription: memcp smart clusterable distributed database\n" \
		> $(DEB_DIR)/DEBIAN/control
	install -m 755 debian/postinst $(DEB_DIR)/DEBIAN/postinst
	install -m 755 debian/prerm    $(DEB_DIR)/DEBIAN/prerm
	install -m 755 debian/postrm   $(DEB_DIR)/DEBIAN/postrm
	echo "/etc/memcp/memcp.conf" > $(DEB_DIR)/DEBIAN/conffiles
	dpkg-deb --build --root-owner-group $(DEB_DIR) memcp.deb
	rm -rf $(DEB_DIR)

docs:
	./memcp -write-docu docs

docker-release:
	sudo docker build -t carli2/memcp:latest .
	sudo docker push carli2/memcp:latest

.PHONY: memcp.sif memcp.deb docs
