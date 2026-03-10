Name:           memcp
Version:        %{_version}
Release:        1%{?dist}
Summary:        memcp smart clusterable distributed database
License:        GPLv3
BuildArch:      %{_arch}

%description
memcp smart clusterable distributed database working best on NVMe.

%pre
getent passwd memcp >/dev/null 2>&1 || \
    useradd -r -s /sbin/nologin -d /var/lib/memcp -c "memcp database daemon" memcp

%install
make -C %{_srcdir} install DESTDIR=%{buildroot} PREFIX=/usr SYSTEMD_DIR=/usr/lib/systemd/system

%post
systemctl daemon-reload
systemctl enable memcp
if systemctl is-active --quiet memcp; then
    systemctl restart memcp
else
    systemctl start memcp
fi

%preun
if [ $1 -eq 0 ]; then
    systemctl stop memcp 2>/dev/null || true
    systemctl disable memcp 2>/dev/null || true
fi

%postun
systemctl daemon-reload 2>/dev/null || true
if [ $1 -eq 0 ]; then
    rm -rf /var/lib/memcp
    rm -rf /etc/memcp
    getent passwd memcp >/dev/null 2>&1 && userdel memcp 2>/dev/null || true
    getent group memcp >/dev/null 2>&1 && groupdel memcp 2>/dev/null || true
fi

%files
/usr/bin/memcp
/usr/lib/memcp/
/usr/lib/systemd/system/memcp.service
%config(noreplace) /etc/memcp/memcp.conf
