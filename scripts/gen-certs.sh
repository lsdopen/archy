#!/bin/bash
set -e

SERVICE=archy-webhook
NAMESPACE=default
# Directory to store certs
CERT_DIR=certs

mkdir -p "$CERT_DIR"

# Optional: Skip if certs exist and are valid? 
# For now, let's regenerate to be safe, or we can check if ca.crt exists.
# echo "Generating certs in $CERT_DIR..."

cat <<EOF > "$CERT_DIR/csr.conf"
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${SERVICE}
DNS.2 = ${SERVICE}.${NAMESPACE}
DNS.3 = ${SERVICE}.${NAMESPACE}.svc
EOF

# Generate CA
openssl genrsa -out "$CERT_DIR/ca.key" 2048
openssl req -x509 -new -nodes -key "$CERT_DIR/ca.key" -days 10000 -out "$CERT_DIR/ca.crt" -subj "/CN=Admission Controller Webhook CA"

# Generate Server Cert
openssl genrsa -out "$CERT_DIR/tls.key" 2048
openssl req -new -key "$CERT_DIR/tls.key" -out "$CERT_DIR/tls.csr" -subj "/CN=${SERVICE}.${NAMESPACE}.svc" -config "$CERT_DIR/csr.conf"

openssl x509 -req -in "$CERT_DIR/tls.csr" -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca.key" -CAcreateserial -out "$CERT_DIR/tls.crt" -days 10000 -extensions v3_req -extfile "$CERT_DIR/csr.conf"

echo "Certificates generated in $CERT_DIR"
