
echo "generate server key and certificate signed with ca.key (Central Authority).."

openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 \
  -CA ca.crt -CAkey ca.key -nodes -keyout server.key -out server.crt -subj "/CN=kotlinlang.ru" \
  -addext "subjectAltName=DNS:kotlinlang.ru,DNS:*.kotlinlang.ru,IP:146.185.138.41" \
  -addext "basicConstraints=CA:FALSE"


