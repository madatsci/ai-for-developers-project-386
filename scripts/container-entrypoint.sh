#!/bin/sh
set -eu

standalone_pgdata=""
standalone_database_started="false"

stop_standalone_database() {
    if [ "$standalone_database_started" = "true" ]; then
        pg_ctl -D "$standalone_pgdata" -m fast -w stop >/dev/null
    fi
}

if [ -z "${DATABASE_URL:-}" ]; then
    echo "DATABASE_URL is not set; starting an ephemeral PostgreSQL database" >&2
    standalone_pgdata="$(mktemp -d /tmp/calendar-postgres.XXXXXX)"
    initdb -D "$standalone_pgdata" --username=calendar --auth=trust --encoding=UTF8 --no-locale >/dev/null
    trap stop_standalone_database EXIT INT TERM
    if ! pg_ctl -D "$standalone_pgdata" -l "$standalone_pgdata/postgres.log" -o "-h 127.0.0.1 -p 5432 -k $standalone_pgdata" -w start >/dev/null; then
        cat "$standalone_pgdata/postgres.log" >&2
        exit 1
    fi
    standalone_database_started="true"
    createdb --host=127.0.0.1 --port=5432 --username=calendar calendar
    export DATABASE_URL="postgres://calendar@127.0.0.1:5432/calendar?sslmode=disable"
fi

/app/migrate
/app/seed

trap - EXIT INT TERM
exec /app/api
