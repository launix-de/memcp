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
CONFIG=/etc/memcp/memcp.conf

configure() {
    # Load existing values or defaults
    DATA=/var/lib/memcp
    API_PORT=4321
    ENABLE_API=true
    MYSQL_PORT=3307
    ENABLE_MYSQL=true
    MYSQL_SOCKET=/run/memcp/memcp.sock
    ROOT_PASSWORD=admin

    if [ -f "$CONFIG" ]; then
        while IFS= read -r line; do
            case "$line" in
                -data*)          DATA="${line#-data }" ;;
                --api-port=*)    API_PORT="${line#--api-port=}" ;;
                --disable-api)   ENABLE_API=false ;;
                --mysql-port=*)  MYSQL_PORT="${line#--mysql-port=}" ;;
                --disable-mysql) ENABLE_MYSQL=false ;;
                --mysql-socket=*)MYSQL_SOCKET="${line#--mysql-socket=}" ;;
            esac
        done < "$CONFIG"
    fi

    if [ -t 0 ]; then
        printf "\n=== memcp configuration ===\n\n"

        printf "Data directory [%s]: " "$DATA"
        read ans; [ -n "$ans" ] && DATA="$ans"

        printf "Enable HTTP API? (true/false) [%s]: " "$ENABLE_API"
        read ans
        case "$ans" in true|false) ENABLE_API="$ans" ;; esac

        if [ "$ENABLE_API" = "true" ]; then
            printf "HTTP API port [%s]: " "$API_PORT"
            read ans; [ -n "$ans" ] && API_PORT="$ans"
        fi

        printf "Enable MySQL protocol? (true/false) [%s]: " "$ENABLE_MYSQL"
        read ans
        case "$ans" in true|false) ENABLE_MYSQL="$ans" ;; esac

        if [ "$ENABLE_MYSQL" = "true" ]; then
            printf "MySQL TCP port [%s]: " "$MYSQL_PORT"
            read ans; [ -n "$ans" ] && MYSQL_PORT="$ans"

            printf "MySQL Unix socket path (empty to disable) [%s]: " "$MYSQL_SOCKET"
            read ans
            if [ "$ans" = "-" ]; then
                MYSQL_SOCKET=""
            elif [ -n "$ans" ]; then
                MYSQL_SOCKET="$ans"
            fi
        fi

        printf "root password [admin]: "
        stty -echo 2>/dev/null || true
        read ans
        stty echo 2>/dev/null || true
        printf "\n"
        [ -n "$ans" ] && ROOT_PASSWORD="$ans"
    fi

    # write config (root password is not stored — applied once during DB init below)
    mkdir -p "$(dirname "$CONFIG")"
    cat > "$CONFIG" <<EOF
# memcp daemon configuration
# One CLI argument per line. Lines starting with # are ignored.
# After editing, run: systemctl restart memcp

-data $DATA

--api-port=$API_PORT
EOF
    if [ "$ENABLE_API" = "false" ]; then
        printf -- "--disable-api\n" >> "$CONFIG"
    fi
    printf "\n" >> "$CONFIG"
    if [ "$ENABLE_MYSQL" = "true" ]; then
        printf -- "--mysql-port=%s\n--mysql-socket=%s\n" "$MYSQL_PORT" "$MYSQL_SOCKET" >> "$CONFIG"
    else
        printf -- "--disable-mysql\n--mysql-socket=\n" >> "$CONFIG"
    fi
    chown root:memcp "$CONFIG"
    chmod 640 "$CONFIG"

    # One-time DB initialization: only on fresh installs (no existing data dir)
    if [ ! -d "$DATA/system" ]; then
        printf "Initializing database...\n"
        INIT_PORT=14321
        mkdir -p "$DATA"
        chown memcp:memcp "$DATA"
        runuser -u memcp -- /usr/bin/memcp -data "$DATA" --root-password="$ROOT_PASSWORD" \
            --no-repl --api-port=$INIT_PORT --disable-mysql \
            </dev/null >/tmp/memcp-init.log 2>&1 &
        INIT_PID=$!
        n=0
        while [ $n -lt 30 ]; do
            sleep 1
            n=$((n+1))
            kill -0 $INIT_PID 2>/dev/null || break
            if ss -tlnp 2>/dev/null | grep -q ":$INIT_PORT "; then break; fi
        done
        kill $INIT_PID 2>/dev/null || true
        wait $INIT_PID 2>/dev/null || true
        printf "Database initialized.\n"
    fi
}

configure
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
