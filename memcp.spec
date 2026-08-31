# Copyright (C) 2026 Carl-Philip Haensch
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

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
Requires(pre):  shadow-utils
Requires(post): coreutils
Requires(post): systemd
Requires(post): util-linux
Requires(preun): systemd
Requires(postun): systemd

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
make all GOOS=linux GOARCH=%{_goarch} CGO_ENABLED=0 \
    LDFLAGS="-s -w"

%install
make install-files DESTDIR=%{buildroot} \
    PREFIX=/usr SYSTEMD_DIR=/usr/lib/systemd/system PACKAGE_FORMAT=rpm

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
/usr/bin/memcp
/usr/lib/memcp/
/usr/lib/systemd/system/memcp.service
/usr/share/man/man1/memcp.1.gz
%attr(0640,root,memcp) %config(noreplace) /etc/memcp/memcp.conf

%changelog
* Tue Sep 01 2026 Carl-Philip Haensch <haensch@launix.de> - %{_version}-1
- Harden package initialization, upgrades, and data retention
