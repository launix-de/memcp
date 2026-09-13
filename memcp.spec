# Copyright (C) 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# Go and the externally built PHP runtime are installed stripped. Their source
# input is recorded in the SRPM/SDK release pins, not an empty debug subpackage.
%global debug_package %{nil}

Name:           memcp
Version:        %{_version}
Release:        1%{?dist}
Summary:        Smart clusterable distributed database
License:        GPL-3.0-or-later
URL:            https://github.com/launix-de/memcp
Source0:        %{name}-%{version}.tar.gz
BuildRequires:  golang
BuildRequires:  make
BuildRequires:  python3
BuildRequires:  gcc
# php-devel is the distribution's development baseline. The actual SDK selected
# by _php_config must additionally be PHP >= 8.5, ZTS and shared embed; make
# check-php validates this before compilation. CI builds that released SDK on
# this same distribution, including Imagick, outside the MemCP source tree.
BuildRequires:  php-devel >= 8.5
%{!?_php_config:%global _php_config php-config}
%{!?_php_license_dir:%global _php_license_dir /usr/share/licenses/php}
Requires(pre):  shadow-utils
Requires(post): coreutils
Requires(post): systemd
Requires(post): util-linux
Requires(preun): systemd
Requires(postun): systemd
# The pre-install script creates these identities before installing owned files.
# Fedora's file dependency generator requires the group for memcp.conf.
Provides:       user(memcp)
Provides:       group(memcp)

%description
MemCP is a persistent, column-oriented in-memory database with HTTP and
MySQL-compatible interfaces.

%prep
%setup -q

%pretrans
# Stop an existing daemon before its binary is replaced. The old process must
# finish its graceful shutdown rebuild before the transaction proceeds.
if [ -x /usr/bin/memcp ] && [ -d /run/systemd/system ]; then
    systemctl stop memcp.service >/dev/null 2>&1 || true
fi

%pre
getent group memcp >/dev/null 2>&1 || groupadd -r memcp
getent passwd memcp >/dev/null 2>&1 || \
    useradd -r -g memcp -s /sbin/nologin -d /var/lib/memcp \
        -c "memcp database daemon" memcp

%build
make all GOOS=linux GOARCH=%{_goarch} PHP_CONFIG="%{_php_config}" \
    PHP_RPATH='$$ORIGIN/../lib/memcp/php' BUILD_FLAGS="-trimpath -buildvcs=false -buildmode=pie" LDFLAGS="-s -w"

%install
make install-files DESTDIR=%{buildroot} \
    PREFIX=/usr SYSTEMD_DIR=/usr/lib/systemd/system PACKAGE_FORMAT=rpm
make install-php-runtime DESTDIR=%{buildroot} PREFIX=/usr \
    PHP_CONFIG="%{_php_config}" PHP_LICENSE_DIR="%{_php_license_dir}"

%check
python3 tools/test_packaging.py

%post
chown root:memcp /etc/memcp/memcp.conf
chmod 640 /etc/memcp/memcp.conf

if [ "$1" -eq 1 ]; then
    /usr/lib/memcp/initialize
fi

if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null || true
    systemctl enable memcp.service >/dev/null
    systemctl restart memcp.service >/dev/null
fi

%preun
if [ "$1" -eq 0 ] && [ -d /run/systemd/system ]; then
    systemctl stop memcp.service >/dev/null 2>&1 || true
    systemctl disable memcp.service >/dev/null 2>&1 || true
fi

%postun
if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
fi
if [ "$1" -eq 0 ] && { [ -d /var/lib/memcp ] || [ -e /etc/memcp/initial-root-password ]; }; then
    echo "memcp: database data and initial credentials were preserved" >&2
fi

%files
%license /usr/share/licenses/memcp/LICENSE
%doc /usr/share/doc/memcp/copyright
%doc /usr/share/doc/memcp/README.md
%doc /usr/share/doc/memcp/CHANGELOG.md
%license /usr/share/doc/memcp/php/
/usr/bin/memcp
/usr/lib/memcp/
/usr/lib/systemd/system/memcp.service
/usr/share/man/man1/memcp.1.gz
%attr(0640,root,memcp) %config(noreplace) /etc/memcp/memcp.conf

%changelog
* Tue Sep 01 2026 Carl-Philip Haensch <haensch@launix.de> - %{_version}-1
- Harden package initialization, upgrades, and data retention
