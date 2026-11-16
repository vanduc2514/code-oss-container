#!/bin/sh
# nginx's entrypoint only executes *.sh files, so this wrapper invokes the
# compiled helper. A non-zero exit aborts startup before nginx reads the include.
exec /usr/local/bin/nginx-routes
