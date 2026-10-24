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
PHP_SDK_ROOT ?= $(PHP_CACHE)/8.5.12-$(GOARCH)
override PHP_CACHE := $(abspath $(PHP_CACHE))
override PHP_SDK_ROOT := $(abspath $(PHP_SDK_ROOT))
PHP_PACKAGE_URL := https://pkg.henderkes.com/api/packages/85/debian/pool/php-zts/main
PHP_PACKAGES_amd64 := \
	php-zts-devel_8.5.12-4_amd64.deb:2c7629c4ae94fdb0f1301421ccb700f983e55ac911de2cc187a37fb3f2ea90b0 \
	php-zts-cli_8.5.12-3_amd64.deb:e8bd629ea68a1b4c5d39b2bcf3d0c447bc31b4640e20c3b3923e70a275bfbd45 \
	php-zts-embed_8.5.12-3_amd64.deb:88cb6501bbacc768e34c37ed1b8d0802c8278614d99bcdefac246b5cb73fb61b \
	php-zts-pdo_8.5.12-1_amd64.deb:d8d387d0b1e3b4a7764a4d7e3d3722f8f113edf7fcd9778e863f9a317b85d8f7 \
	php-zts-mysqlnd_8.5.12-1_amd64.deb:4dc0d71b51e57a9bae41bb77dd1c05aa9e86e542b3b9ffaff634dcb86f506da2 \
	php-zts-pdo-mysql_8.5.12-1_amd64.deb:1eb1e948e69239b4f655beb603904de22d2201bf8f79365bfeb9b13c3009418c \
	php-zts-pdo-sqlite_8.5.12-1_amd64.deb:68d126fe281c2655b3b79749c2c5036e39a18432e751a058666ee1a335517514 \
	php-zts-mysqli_8.5.12-1_amd64.deb:e9ceba43e1a90a169d61c9cf35cfed92b71d50328a3e61790e7903062c8dc823 \
	php-zts-gmp_8.5.12-1_amd64.deb:825474bed70abddd3149de3fde6dab58a3b22a85f04020781ef11c1b33273c89 \
	php-zts-gettext_8.5.12-1_amd64.deb:14ec200c1349532add7a23ad9f9f961ed506d920efe0c2cdaf5c9557ca8df959 \
	php-zts-intl_8.5.12-1_amd64.deb:696754d791d28a677b36fc306321e76618c35722786542c37a4d4345b8aee0ba \
	php-zts-gd_8.5.12-1_amd64.deb:680e7427ef023ab333f49ea72b1b69fc4b23c33bf20e7aae62414a2d9b4b31f8 \
	php-zts-zip_1.22.8+php85.12+ext-1_amd64.deb:5f6ee401644e0f5f51605b43a166de10028e740623e82581a509d926c4374e36 \
	php-zts-imagick_3.8.1+php85.12+ext-1_amd64.deb:9daa031d2d9edaf756c84cc827487c93bfc2b9bc99fab237881260f0eaec640d
PHP_PACKAGES_arm64 := \
	php-zts-devel_8.5.12-3_arm64.deb:90920b4555ad0a7f611bc7f7ef5241038b0d388a56bf38e55a4056725bb4843b \
	php-zts-cli_8.5.12-3_arm64.deb:e4e2a74d86a33285504841c6e8b351502fe8b23b3f67e7dbbfaf4c9710a56a7d \
	php-zts-embed_8.5.12-3_arm64.deb:091892ade4cba10b3078bbb812b924ccf40d57611b7e2657b6533b11eadb2f56 \
	php-zts-pdo_8.5.12-1_arm64.deb:8f3d6d2d202cb20e404312c429b7de4c3490d30d128bac1c6f555394d9371fa2 \
	php-zts-mysqlnd_8.5.12-1_arm64.deb:3bc7cf3a8e1c271c5ea293c816719b6e3f656607fd8b28552075e5298bdd65ea \
	php-zts-pdo-mysql_8.5.12-1_arm64.deb:c939d611e95f5600588def06f38c0a64920d9d42569d991ab2da9c14f975103b \
	php-zts-pdo-sqlite_8.5.12-1_arm64.deb:8766665836e2909484e846f9ae82befd3e7f9615395797973e72c3301e368440 \
	php-zts-mysqli_8.5.12-1_arm64.deb:0a53201a01393fa1263f061eca3c84f967a8c700848134bbaa2131dd6254e91a \
	php-zts-gmp_8.5.12-1_arm64.deb:8a1374ae738c74f47bb5a5e67de046cfa6e4d40b8e1841e7f1bc55759f2565c2 \
	php-zts-gettext_8.5.12-1_arm64.deb:92b78532db2db745e6c1253247aadb818e92e673b42202c324825c46832de6d9 \
	php-zts-intl_8.5.12-1_arm64.deb:80b4b461f7087afd13cb590a568e7f8b1ca2c50f7d5cff11a09aac2596d9a3a6 \
	php-zts-gd_8.5.12-1_arm64.deb:decb8a7abc5954b29a88696d57ff10849150007399d8f94bc13dbb2085146c54 \
	php-zts-zip_1.22.8+php85.12+ext-1_arm64.deb:ec2df81965ff180b0ecfd18cbb7269f2f607d57da5e15250a35e25a2f2782e0e \
	php-zts-imagick_3.8.1+php85.12+ext-1_arm64.deb:0bb667e2eed0c15bd6f61e4a9b3c70f373ee2827c8c47ca7a9b66977cfad21ac
# Explicit PHP_CONFIG wins. Reuse a complete installed ZTS SDK where present;
# an ordinary NTS php-dev package cannot be used by the embedded host.
ifeq ($(origin PHP_CONFIG), undefined)
PHP_CONFIG := $(shell for pc in php-config "$(HOME)/.local/opt/memcp-php/bin/php-config"; do \
 command -v "$$pc" >/dev/null 2>&1 || continue; \
 test "$$($$pc --vernum 2>/dev/null)" -ge 80512 2>/dev/null || continue; \
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
# Only the downloaded SDK needs its separately linked PDO module on RUNPATH.
# Keep the existing installed-SDK build flags unchanged.
PHP_RPATH    ?= $(shell $(PHP_CONFIG) --prefix)/lib$(if $(PHP_CACHED),:$(shell $(PHP_CONFIG) --extension-dir))
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
	@test "$$($(PHP_CONFIG) --vernum)" -ge 80512 || { echo "PHP >= 8.5.12 ZTS SDK required (CVE-2026-109618, CVE-2026-109619)" >&2; exit 1; }
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

# Cross execution uses static user-mode QEMU and needs no target C libraries.
# Install qemu-user-static; override JIT_CROSS_ARCH and JIT_CROSS_EMULATOR to
# select another supported Linux target.
JIT_CROSS_ARCH ?= arm64
JIT_CROSS_EMULATOR_arm64 := qemu-aarch64-static
JIT_CROSS_EMULATOR_riscv64 := qemu-riscv64-static
JIT_CROSS_EMULATOR ?= $(JIT_CROSS_EMULATOR_$(JIT_CROSS_ARCH))
jit-cross-test: jit-toolchain
	@test -n "$(JIT_CROSS_EMULATOR)" || { echo "Use arm64 or riscv64 for JIT_CROSS_ARCH" >&2; exit 1; }
	@command -v "$(JIT_CROSS_EMULATOR)" >/dev/null
	CGO_ENABLED=0 GOOS=linux GOARCH=$(JIT_CROSS_ARCH) \
		GOROOT="$(JIT_GOROOT)" GOEXPERIMENT=jit "$(JIT_GOROOT)/bin/go" \
		test -tags=nophp -exec="$(JIT_CROSS_EMULATOR)" ./scm ./storage

jitgen:
	@set -eu; \
	jitgen_bin=$$(mktemp /tmp/memcp-jitgen.XXXXXX); \
	trap 'rm -f "$$jitgen_bin"' EXIT; \
	GOOS=$$(go env GOHOSTOS) GOARCH=$$(go env GOHOSTARCH) go build -o "$$jitgen_bin" ./tools/jitgen/; \
	"$$jitgen_bin" -patch scm/alu.go scm/compare.go scm/list.go scm/strings.go scm/scm.go scm/date.go scm/streams.go scm/sync.go scm/metrics.go scm/scheduler.go scm/window.go scm/vector.go scm/packrat.go scm/jit.go scm/timezone.go scm/processlist.go scm/list_assoc_extra.go scm/expression_name.go scm/json_functions.go; \
	"$$jitgen_bin" -patch storage/storage-int.go storage/storage-float.go storage/storage-decimal.go storage/storage-string.go storage/storage-prefix.go storage/storage-enum.go storage/storage-scmer.go storage/storage-sparse.go storage/storage-seq.go storage/storage-const.go storage/overlay-blob.go storage/compute_proxy.go storage/jit_getters.go; \
	gofmt -w scm storage

jitgen-policy:
	@set -eu; \
	jitgen_bin=$$(mktemp /tmp/memcp-jitgen.XXXXXX); \
	trap 'rm -f "$$jitgen_bin"' EXIT; \
	GOOS=$$(go env GOHOSTOS) GOARCH=$$(go env GOHOSTARCH) go build -o "$$jitgen_bin" ./tools/jitgen/; \
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
	# Full-suite CI entrypoint; use focused tests during local development.
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
