#!/bin/sh
set -e

CERT_DIR=/etc/nginx/ssl
CERT_KEY=$CERT_DIR/nginx-selfsigned.key
CERT_CRT=$CERT_DIR/nginx-selfsigned.crt

if [ ! -f "$CERT_KEY" ] || [ ! -f "$CERT_CRT" ]; then
    echo "Generating self-signed SSL certificate..."
    mkdir -p "$CERT_DIR"
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "$CERT_KEY" \
        -out "$CERT_CRT" \
        -subj "/CN=localhost" 2>/dev/null
    echo "Self-signed certificate generated."
fi

exec nginx -g "daemon off;"