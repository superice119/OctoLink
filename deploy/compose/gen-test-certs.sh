#!/usr/bin/env bash
# Generates self-signed test TLS certs for NATS. NOT for production use.
set -euo pipefail

CERTS_DIR="${1:-./nats_config}"
mkdir -p "$CERTS_DIR"

echo "Generating test TLS certificates in $CERTS_DIR ..."

# Root CA
openssl genrsa -out "$CERTS_DIR/rootCA.key" 4096
openssl req -x509 -new -nodes -key "$CERTS_DIR/rootCA.key" \
  -sha256 -days 3650 \
  -subj "/CN=OctoLink-Test-CA/O=OctoLink/C=CN" \
  -out "$CERTS_DIR/rootCA.pem"

# Server cert with Subject Alternative Names (required for TLS clients that
# connect using the Docker service name "msg_broker" instead of "nats").
openssl genrsa -out "$CERTS_DIR/key.pem" 2048
openssl req -new -key "$CERTS_DIR/key.pem" \
  -subj "/CN=nats/O=OctoLink/C=CN" \
  -out "$CERTS_DIR/server.csr"

# Write SAN extension file
cat > "$CERTS_DIR/v3_ext.cnf" <<EOF
[req]
req_extensions = v3_req
[v3_req]
subjectAltName = DNS:msg_broker,DNS:nats,DNS:localhost,IP:127.0.0.1
EOF

openssl x509 -req -in "$CERTS_DIR/server.csr" \
  -CA "$CERTS_DIR/rootCA.pem" -CAkey "$CERTS_DIR/rootCA.key" \
  -CAcreateserial -out "$CERTS_DIR/cert.pem" \
  -days 825 -sha256 \
  -extfile "$CERTS_DIR/v3_ext.cnf" -extensions v3_req

rm -f "$CERTS_DIR/server.csr" "$CERTS_DIR/rootCA.key" "$CERTS_DIR/rootCA.srl" "$CERTS_DIR/v3_ext.cnf"

echo "Done. Files generated:"
echo "  $CERTS_DIR/rootCA.pem"
echo "  $CERTS_DIR/cert.pem"
echo "  $CERTS_DIR/key.pem"
echo ""
echo "WARNING: These are TEST certificates only. Do NOT use in production."
