
echo "generate server key and certificate signed with ca.key (Central Authority).."

openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 \
  -CA ca.crt -CAkey ca.key -nodes -keyout server.key -out server.crt -subj "/CN=example.com" \
  -addext "subjectAltName=DNS:example.com,DNS:*.example.com,IP:1.2.3.4" \
  -addext "basicConstraints=CA:FALSE"


