#!/bin/sh
set -eu

/app/migrate
/app/seed
exec /app/api
