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
# Native precompiled PHP 8.5 ZTS/embed packages, published by the package
# provider documented at https://frankenphp.dev/docs/. No PHP source build,
# sudo, or extension development libraries are needed. Keep version/checksum
# pairs together. All downloaded files stay in the ignored .third_party cache.
PHP_CACHE ?= $(CURDIR)/.third_party/php
PHP_SDK_ROOT ?= $(PHP_CACHE)/8.5.11-$(GOARCH)
override PHP_CACHE := $(abspath $(PHP_CACHE))
override PHP_SDK_ROOT := $(abspath $(PHP_SDK_ROOT))
PHP_PACKAGE_URL := https://pkg.henderkes.com/api/packages/85/debian/pool/php-zts/main
PHP_PACKAGES_amd64 := \
	php-zts-devel_8.5.11-1_amd64.deb:0b0e6b6a77879c66460393a1acbc510a551cb9e0093937c4bf75a9b13f68f08d \
	php-zts-cli_8.5.11-1_amd64.deb:71230fcc152097a64b46b7feefc086b5ed7d900521d095b6517391eba9193dfb \
	php-zts-embed_8.5.11-1_amd64.deb:e354af7e2a1d13a0962c8c15aefb3fd67fc53b216264361029ab962888c82092 \
	php-zts-pdo_8.5.11-1_amd64.deb:d2805258ed22cef34f5fd8db9897284f9fb1d169d849531ac211b2bca88a6380 \
	php-zts-mysqlnd_8.5.11-1_amd64.deb:709cbdbb129a4a132f4e39748ef2e61e889b609541432bd0ed41a427004143f4 \
	php-zts-pdo-mysql_8.5.11-1_amd64.deb:794d9db5079fe5e2613e09d97033dab6c269cde407302528484948678972f85d \
	php-zts-pdo-sqlite_8.5.11-1_amd64.deb:482ae8503f9322b492da4ee076a4baffc7b83dee7b003c97d86674a1e019c79c \
	php-zts-mysqli_8.5.11-1_amd64.deb:6ea3da2c6139b576229ec76b7edc3eb6a9965334979a6d1f987effa3bda6340b \
	php-zts-gmp_8.5.11-1_amd64.deb:690792d179857390177eba5dd35b0536e27d4fa4f74f3b038925d9a8600dce58 \
	php-zts-gettext_8.5.11-1_amd64.deb:7414f6152e6018bec2d39de0bc333d30523e60bd4924a639cb0dad33551fdb9d \
	php-zts-intl_8.5.11-1_amd64.deb:b8e4c34dbfd516b731e22f431291c314bc6a03c94a2974b85ae9e0a2b2570bd7 \
	php-zts-gd_8.5.11-1_amd64.deb:c409e0078bf3a398fecd2f62b7bbdd5492d39646001317d0906f58697bd4bfe7 \
	php-zts-zip_1.22.8+php85-2_amd64.deb:65a3a344506d151a165203017d78fd8046371fb66da30955d4f925012ed4ac47 \
	php-zts-imagick_3.8.1+php85-9_amd64.deb:03df6d1d94f4bc0743097afd8f11fc5051e717b0137deb5122e3d3f131ecc30b
PHP_PACKAGES_arm64 := \
	php-zts-devel_8.5.11-1_arm64.deb:bb8a5d96beed29888064ca34428c98b89e9f4c11d606e3276f3d3c9f17d508d2 \
	php-zts-cli_8.5.11-1_arm64.deb:cfda8a42c416319545689c643115e931801851b0212ae825aa826e5409c81ab9 \
	php-zts-embed_8.5.11-1_arm64.deb:04f8616206cd7b44739c4430687f1a693cba8c6281530d43d6feef182eb3be14 \
	php-zts-pdo_8.5.11-1_arm64.deb:927e08792b30d56a54cc7aa1495945420b69e45283d2a673b9dc5d5b40f4f07a \
	php-zts-mysqlnd_8.5.11-1_arm64.deb:b5091ce23de84a1df91efaa3372654dcb71732529b4297d9ad38f8a04d5a5adf \
	php-zts-pdo-mysql_8.5.11-1_arm64.deb:153768eeec6e3d89404eee23d5917589d981069267ce4ea94a483f369e2483aa \
	php-zts-pdo-sqlite_8.5.11-1_arm64.deb:e93bcf64434df142560844d37e0220c474decd94494bbd0e33c09d39e41c62e9 \
	php-zts-mysqli_8.5.11-1_arm64.deb:906bc005d666175e30b97024f39fc3ab06a4706dab5bea910b95fdc2d90c7aa7 \
	php-zts-gmp_8.5.11-1_arm64.deb:07b311c780b25b0b9abd37d6b3f2a5e742301d66eb0511dc80af2dd7e14c14aa \
	php-zts-gettext_8.5.11-1_arm64.deb:5229999d12a3f816353db2eb603a2e20181cac824653d48ca56483ff13b7a1f0 \
	php-zts-intl_8.5.11-1_arm64.deb:cf8475896161a4219406e9105c2c803a57e233066e13456a52d5e1c972975a94 \
	php-zts-gd_8.5.11-1_arm64.deb:5dae6e98ce25ff821cedf5937b8875c2b0b3f10a916df0fe50c58c1fbf63a203 \
	php-zts-zip_1.22.8+php85-2_arm64.deb:d1b7b6964c00f27e16923fdd11609030879a0dee67639df76c26e7fb8ff2545f \
	php-zts-imagick_3.8.1+php85-9_arm64.deb:1f577fa9cd6126381d992a95fe12197e52d611a258696d3f6e9ae936ff8b16f5
# Explicit PHP_CONFIG wins. Reuse a complete installed ZTS SDK where present;
# an ordinary NTS php-dev package cannot be used by the embedded host.
ifeq ($(origin PHP_CONFIG), undefined)
PHP_CONFIG := $(shell for pc in php-config "$(HOME)/.local/opt/memcp-php/bin/php-config"; do \
 command -v "$$pc" >/dev/null 2>&1 || continue; \
 test "$$($$pc --vernum 2>/dev/null)" -ge 80500 2>/dev/null || continue; \
 "$$pc" --configure-options | grep -q -- --enable-zts || continue; \
 test -f "$$($$pc --prefix)/lib/libphp.so" || continue; \
 "$$($$pc --php-binary)" -r 'exit(PHP_ZTS && extension_loaded("pdo") && extension_loaded("imagick") && extension_loaded("gd") && extension_loaded("zip") ? 0 : 1);' >/dev/null 2>&1 || continue; \
 command -v "$$pc"; break; done)
ifeq ($(PHP_CONFIG),)
PHP_CONFIG = $(PHP_SDK_ROOT)/bin/php-config
endif
endif
PHP_CACHED = $(filter $(PHP_SDK_ROOT)/bin/php-config,$(abspath $(PHP_CONFIG)))
PHP_PROVISION = $(if $(PHP_CACHED),php-toolchain)
# Use the host's existing packaged-runtime layout for downloaded extensions.
# The public binary names remain ./memcp and ./memcp-php (symlinks to ELF files).
PHP_BINARY_DIR = $(if $(PHP_CACHED),$(CURDIR)/.build/php/bin,.)
PHP_RUNTIME = $(if $(PHP_CACHED),php-runtime)
PHP_TAGS     := php,nowatcher,nobrotli,nomercure
PHP_RPATH    ?= $(shell $(PHP_CONFIG) --prefix)/lib:$(shell $(PHP_CONFIG) --extension-dir)
PHP_ENV      = CGO_ENABLED=1 CGO_CFLAGS="$$($(PHP_CONFIG) --includes)" CGO_LDFLAGS="-L$$($(PHP_CONFIG) --prefix)/lib "'-Wl,-rpath,$(PHP_RPATH)'" $$($(PHP_CONFIG) --ldflags) $$($(PHP_CONFIG) --libs)"
PHP_LICENSE_DIR ?= $(shell $(PHP_CONFIG) --prefix)/share/licenses/php
PACKAGE_PHP_RPATH = '$$$$ORIGIN/../lib/memcp/php:$$$$ORIGIN/../lib/memcp/php/extensions'
STRIP ?= strip
PACKAGE_LDFLAGS ?= -s -w
DIST_DIR     ?= dist
PACKAGE_DIR  ?= .build/packages
SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct)
JIT_GOROOT   ?= $(CURDIR)/.third_party/go-jit
JIT_GO_REPOSITORY ?= https://github.com/launix-de/go.git
JIT_GO_REF   ?= jit-foreign-frames-go1.27.0
export SOURCE_DATE_EPOCH

all: check-php $(PHP_RUNTIME)
	$(PHP_ENV) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) -tags=$(PHP_TAGS) -ldflags="$(LDFLAGS)" -o $(PHP_BINARY_DIR)/memcp .
	$(if $(PHP_CACHED),ln -sfn $(PHP_BINARY_DIR)/memcp memcp,@:)

nophp:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(BUILD_FLAGS) -tags=nophp -ldflags="$(LDFLAGS)" -o memcp .

# Fail before compiling when php-config points at an NTS/FPM-only SDK.
.PHONY: check-php
check-php: $(PHP_PROVISION)
	@test "$$($(PHP_CONFIG) --vernum)" -ge 80500 || { echo "PHP >= 8.5 ZTS SDK required" >&2; exit 1; }
	@case "$$($(PHP_CONFIG) --configure-options)" in *--enable-zts*) ;; *) echo "PHP ZTS SDK required" >&2; exit 1 ;; esac
	@test -f "$$($(PHP_CONFIG) --prefix)/lib/libphp.so" || { echo "PHP embed shared library required" >&2; exit 1; }

# Fetch and verify released binaries; serialize concurrent make invocations.
.PHONY: php-toolchain php-runtime
php-toolchain:
	@set -eu; \
	case "$(GOOS):$(GOARCH):$$(uname -m)" in linux:amd64:x86_64|linux:arm64:aarch64) ;; \
		*) echo "Use PHP_CONFIG for this platform/cross build (binary SDK: native Linux amd64/arm64)." >&2; exit 1 ;; esac; \
	getconf GNU_LIBC_VERSION >/dev/null 2>&1 || { echo "Binary PHP SDK requires glibc." >&2; exit 1; }; \
	mkdir -p "$(PHP_CACHE)"; \
	exec 9>"$(PHP_CACHE)/.sdk.lock"; flock 9; \
	expected=$$(printf '%s\n' '$(PHP_PACKAGES_$(GOARCH)):1' | sha256sum | cut -d' ' -f1); \
	if [ -x "$(PHP_SDK_ROOT)/bin/php-config" ] && [ -f "$(PHP_SDK_ROOT)/usr/lib/libphp.so" ] \
		&& [ "$$(cat "$(PHP_SDK_ROOT)/.complete" 2>/dev/null)" = "$$expected" ]; then \
		echo "Using cached PHP SDK: $(PHP_SDK_ROOT)"; exit 0; \
	fi; \
	for tool in curl dpkg-deb; do command -v "$$tool" >/dev/null || { echo "Missing $$tool for binary SDK extraction" >&2; exit 1; }; done; \
	test ! -e "$(PHP_SDK_ROOT)" || { echo "Incomplete/different SDK at $(PHP_SDK_ROOT); select a fresh PHP_SDK_ROOT." >&2; exit 1; }; \
	stage=$$(mktemp -d "$(PHP_CACHE)/.sdk.XXXXXX"); \
	trap 'rm -rf "$$stage"' EXIT HUP INT TERM; \
	mkdir -p "$(PHP_CACHE)/downloads" "$$stage/bin"; \
	for package in $(PHP_PACKAGES_$(GOARCH)); do \
		name=$${package%:*}; checksum=$${package##*:}; archive="$(PHP_CACHE)/downloads/$$name"; \
		if ! echo "$$checksum  $$archive" | sha256sum --check --status 2>/dev/null; then \
			echo "Downloading $$name"; \
			curl -fL --retry 3 "$(PHP_PACKAGE_URL)/$$name" -o "$$stage/download"; \
			echo "$$checksum  $$stage/download" | sha256sum --check; \
			mv "$$stage/download" "$$archive"; \
		fi; \
		dpkg-deb --extract "$$archive" "$$stage"; \
	done; \
	ln -s libphp-zts-85.so "$$stage/usr/lib/libphp.so"; \
	ln -s imagick-zts-85.so "$$stage/usr/lib/php-zts/modules/imagick.so"; \
	ln -s php-zts "$$stage/usr/share/licenses/php"; \
	printf '%s\n' '#!/bin/sh' \
		'# Copyright (C) 2026 Carl-Philip Haensch; SPDX-License-Identifier: GPL-3.0-or-later' \
		'set -eu' 'prefix=$$(CDPATH= cd -- "$$(dirname -- "$$0")/../usr" && pwd)' \
		'case "$${1:-}" in' '--prefix) echo "$$prefix" ;;' \
		'--includes) for dir in "" /main /TSRM /Zend /ext /ext/date/lib; do printf -- "-I%s/include/php-zts%s " "$$prefix" "$$dir"; done; echo ;;' \
		'--extension-dir) echo "$$prefix/lib/php-zts/modules" ;;' \
		'--ldflags) echo "-L$$prefix/lib -L$$prefix/lib/php-zts/modules -lpthread" ;;' \
		'--libs) echo "-l:pdo-zts-85.so" ;;' \
		'--php-binary) echo "$$prefix/bin/php-zts" ;;' \
		'*) exec "$$prefix/bin/php-config-zts" "$$@" ;;' 'esac' > "$$stage/bin/php-config"; \
	chmod +x "$$stage/bin/php-config"; \
	for module in pdo mysqlnd pdo_mysql pdo_sqlite mysqli gmp gettext intl gd zip; do \
		printf 'extension=%s-zts-85.so\n' "$$module"; \
	done > "$$stage/usr/lib/memcp-extensions.ini"; \
	"$$stage/usr/bin/php-zts" -n -d "extension_dir=$$stage/usr/lib/php-zts/modules" \
		-d extension=pdo-zts-85.so -d extension=imagick-zts-85.so -d extension=gd-zts-85.so -d extension=zip-zts-85.so \
		-r 'exit(PHP_ZTS && extension_loaded("pdo") && extension_loaded("imagick") && extension_loaded("gd") && extension_loaded("zip") ? 0 : 1);'; \
	printf '%s\n' "$$expected" > "$$stage/.complete"; \
	mv "$$stage" "$(PHP_SDK_ROOT)"

# The existing host discovers ../lib/memcp/php relative to its ELF executable.
# Stage that private layout; no global PHPRC, system install, or host code edits.
php-runtime: check-php
	@mkdir -p "$(PHP_BINARY_DIR)" "$(CURDIR)/.build/php/lib/memcp/php/conf.d"
	ln -sfn "$$($(PHP_CONFIG) --prefix)/lib/php-zts/modules" "$(CURDIR)/.build/php/lib/memcp/php/extensions"
	cp packaging/php.ini "$(CURDIR)/.build/php/lib/memcp/php/php.ini"
	cp "$$($(PHP_CONFIG) --prefix)/lib/memcp-extensions.ini" "$(CURDIR)/.build/php/lib/memcp/php/conf.d/00-sdk.ini"

# libphp and its extensions are external dependencies. Use a matching ZTS
# php-config; optional FrankenPHP services are excluded from this host.
.PHONY: php memcp-php test-php nophp
memcp-php: php
php: check-php $(PHP_RUNTIME)
	$(PHP_ENV) \
		go build $(BUILD_FLAGS) -tags=$(PHP_TAGS) -ldflags="$(LDFLAGS)" -o $(PHP_BINARY_DIR)/memcp-php .
	$(if $(PHP_CACHED),ln -sfn $(PHP_BINARY_DIR)/memcp-php memcp-php,@:)

test-php: php
	$(PHP_ENV) \
		go test -tags=$(PHP_TAGS) -run 'TestPHP|TestResultBuffer|TestCatalog|TestIMAP' -count=1 -v . ./phpbridge

# Keep the experimental compiler outside the tracked source tree. Clean
# checkouts fast-forward on every invocation, while a checkout with local
# tracked changes is left untouched so compiler work is never discarded.
jit-toolchain:
	@set -eu; \
	if [ ! -e "$(JIT_GOROOT)" ]; then \
		mkdir -p "$(dir $(JIT_GOROOT))"; \
		git clone --depth 1 --branch "$(JIT_GO_REF)" "$(JIT_GO_REPOSITORY)" "$(JIT_GOROOT)"; \
	else \
		test -d "$(JIT_GOROOT)/.git" || { echo "$(JIT_GOROOT) is not a Go git checkout" >&2; exit 1; }; \
		if git -C "$(JIT_GOROOT)" diff --quiet && git -C "$(JIT_GOROOT)" diff --cached --quiet; then \
			git -C "$(JIT_GOROOT)" pull --ff-only origin "$(JIT_GO_REF)"; \
		else \
			echo "warning: $(JIT_GOROOT) has local changes; skipping compiler update" >&2; \
		fi; \
	fi; \
	test -d "$(JIT_GOROOT)/.git" || { echo "$(JIT_GOROOT) is not a Go git checkout" >&2; exit 1; }; \
	revision=$$(git -C "$(JIT_GOROOT)" rev-parse HEAD); \
	built_revision=$$(cat "$(JIT_GOROOT)/.memcp-built-revision" 2>/dev/null || true); \
	if [ ! -x "$(JIT_GOROOT)/bin/go" ] || [ "$$built_revision" != "$$revision" ]; then \
		version=$$(sed -n 's/^go\([0-9].*\)$$/\1/p' "$(JIT_GOROOT)/VERSION" | head -n 1); \
		test -n "$$version"; \
		bootstrap_goroot=$$(GOTOOLCHAIN="go$$version" go env GOROOT); \
		(cd "$(JIT_GOROOT)/src" && GOROOT_BOOTSTRAP="$$bootstrap_goroot" ./make.bash); \
		printf '%s\n' "$$revision" > "$(JIT_GOROOT)/.memcp-built-revision"; \
	fi

jit: check-php $(PHP_RUNTIME) jit-toolchain
	$(PHP_ENV) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		GOROOT="$(JIT_GOROOT)" GOEXPERIMENT=jit "$(JIT_GOROOT)/bin/go" \
		build $(BUILD_FLAGS) -tags=$(PHP_TAGS) -ldflags="$(LDFLAGS)" -o $(PHP_BINARY_DIR)/memcp .
	$(if $(PHP_CACHED),ln -sfn $(PHP_BINARY_DIR)/memcp memcp,@:)

jit-nophp: jit-toolchain
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		GOROOT="$(JIT_GOROOT)" GOEXPERIMENT=jit "$(JIT_GOROOT)/bin/go" \
		build $(BUILD_FLAGS) -tags=nophp -ldflags="$(LDFLAGS)" -o memcp .

jitgen:
	@set -eu; \
	jitgen_bin=$$(mktemp /tmp/memcp-jitgen.XXXXXX); \
	trap 'rm -f "$$jitgen_bin"' EXIT; \
	go build -o "$$jitgen_bin" ./tools/jitgen/; \
	"$$jitgen_bin" -patch scm/alu.go scm/compare.go scm/list.go scm/strings.go scm/scm.go scm/date.go scm/streams.go scm/sync.go scm/metrics.go scm/scheduler.go scm/window.go scm/vector.go scm/packrat.go scm/jit.go scm/timezone.go scm/processlist.go scm/list_assoc_extra.go scm/expression_name.go scm/json_functions.go; \
	"$$jitgen_bin" -patch storage/storage-int.go storage/storage-float.go storage/storage-decimal.go storage/storage-string.go storage/storage-prefix.go storage/storage-enum.go storage/storage-scmer.go storage/storage-sparse.go storage/storage-seq.go storage/storage-const.go storage/overlay-blob.go storage/compute_proxy.go storage/jit_getters.go; \
	gofmt -w scm storage

jitgen-policy:
	@set -eu; \
	jitgen_bin=$$(mktemp /tmp/memcp-jitgen.XXXXXX); \
	trap 'rm -f "$$jitgen_bin"' EXIT; \
	go build -o "$$jitgen_bin" ./tools/jitgen/; \
	"$$jitgen_bin" -patch -policy-only scm/alu.go scm/compare.go scm/list.go scm/strings.go scm/scm.go scm/date.go scm/streams.go scm/sync.go scm/metrics.go scm/scheduler.go scm/window.go scm/vector.go scm/packrat.go scm/jit.go scm/timezone.go scm/processlist.go scm/list_assoc_extra.go scm/expression_name.go scm/json_functions.go; \
	gofmt -w scm

costgen:
	go run ./tools/costgen -patch

ceph:
	go build -tags=ceph

install: all install-files

# Only the released, externally built runtime is installed; never PHP sources.
install-php-runtime: check-php
	install -d $(DESTDIR)$(PREFIX)/lib/memcp/php/extensions $(DESTDIR)$(PREFIX)/lib/memcp/php/conf.d $(DESTDIR)$(PREFIX)/share/doc/memcp/php
	install -m 644 "$$($(PHP_CONFIG) --prefix)/lib/libphp.so" $(DESTDIR)$(PREFIX)/lib/memcp/php/libphp.so
	test -f "$$($(PHP_CONFIG) --extension-dir)/imagick.so"
	install -m 644 "$$($(PHP_CONFIG) --extension-dir)"/*.so $(DESTDIR)$(PREFIX)/lib/memcp/php/extensions/
	$(STRIP) --strip-unneeded $(DESTDIR)$(PREFIX)/lib/memcp/php/libphp.so $(DESTDIR)$(PREFIX)/lib/memcp/php/extensions/*.so
	install -m 644 packaging/php.ini $(DESTDIR)$(PREFIX)/lib/memcp/php/php.ini
	@if [ -f "$$($(PHP_CONFIG) --prefix)/lib/memcp-extensions.ini" ]; then \
		ln -sf libphp.so $(DESTDIR)$(PREFIX)/lib/memcp/php/libphp-zts-85.so; \
		install -m 644 "$$($(PHP_CONFIG) --prefix)/lib/memcp-extensions.ini" $(DESTDIR)$(PREFIX)/lib/memcp/php/conf.d/00-sdk.ini; \
	fi
	test -f "$(PHP_LICENSE_DIR)/LICENSE"
	cd "$(PHP_LICENSE_DIR)" && find . -type f \( -name LICENSE -o -name 'LICENSE.*' -o -name COPYING -o -name 'COPYING.*' -o -name NOTICE -o -name '*LICENSE*.txt' \) \
		-exec install -D -m 644 '{}' '$(abspath $(DESTDIR)$(PREFIX)/share/doc/memcp/php)/{}' \;

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
	MEMCP_FAIL_FAST=0 MEMCP_COVERAGE=1 MEMCP_COVERDIR=$${MEMCP_COVERDIR:-/tmp/memcp-coverage} ./git-pre-commit

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
	$(MAKE) all GOOS=linux GOARCH=$(DEB_GOARCH) PHP_RPATH=$(PACKAGE_PHP_RPATH) BUILD_FLAGS="$(BUILD_FLAGS) -buildmode=pie" LDFLAGS="$(PACKAGE_LDFLAGS)"
	rm -rf -- $(DEB_DIR)
	mkdir -p $(DEB_DIR)/DEBIAN
	$(MAKE) install-files DESTDIR=$(DEB_DIR) PREFIX=/usr SYSTEMD_DIR=/usr/lib/systemd/system PACKAGE_FORMAT=deb
	$(MAKE) install-php-runtime DESTDIR=$(DEB_DIR) PREFIX=/usr
	@set -eu; \
	deps=$$(dpkg-shlibdeps -O -I$(DEB_DIR)/usr/lib/memcp/php -l$(DEB_DIR)/usr/lib/memcp/php -e$(DEB_DIR)/usr/bin/memcp -e$(DEB_DIR)/usr/lib/memcp/php/libphp.so $(DEB_DIR)/usr/lib/memcp/php/extensions/*.so); \
	deps=$${deps#shlibs:Depends=}; test -n "$$deps"; \
	printf "Package: memcp\nVersion: $(VERSION)\nArchitecture: $(DEB_ARCH)\nSection: database\nPriority: optional\nMaintainer: Carl-Philip Hänsch <hänsch@launix.de>\nDepends: adduser, %s\nHomepage: https://github.com/launix-de/memcp\nDescription: smart clusterable distributed database\n MemCP is a persistent, column-oriented in-memory database with HTTP and\n MySQL-compatible interfaces.\n" \
		"$$deps" > $(DEB_DIR)/DEBIAN/control
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
	git archive --format=tar --prefix=memcp-$(VERSION)/ \
		-o $(PACKAGE_DIR)/rpmbuild/SOURCES/memcp-$(VERSION).tar $(SOURCE_TREEISH)
	gzip -9n $(PACKAGE_DIR)/rpmbuild/SOURCES/memcp-$(VERSION).tar
	rpmbuild $(RPMBUILD_FLAGS) -ba memcp.spec \
		--target "$(RPM_ARCH)" \
		--define "_topdir $(PWD)/$(PACKAGE_DIR)/rpmbuild" \
		--define "_version $(VERSION)" \
		--define "_goarch $(RPM_GOARCH)" \
		--define "_php_config $(PHP_CONFIG)" --define "_php_license_dir $(PHP_LICENSE_DIR)"
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

.PHONY: all jit jit-toolchain install install-files memcp.sif memcp.deb memcp.rpm \
	package-check version artifact-names docs docker-release jitgen costgen
