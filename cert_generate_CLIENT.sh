
echo "generate client's certificate.."

CA=ca
NAME=client

openssl req -x509 -newkey rsa:4096 \
    -CA $CA.crt -CAkey $CA.key \
    -keyout $NAME.key -out $NAME.crt -days 3650 -passin pass:1234 -passout pass:1234 \
    -addext "basicConstraints=CA:FALSE"

echo "export to the normal format (for browsers etc).."
openssl pkcs12 -export -out $NAME.p12 -inkey $NAME.key -in $NAME.crt -passin pass:1234 -passout pass:1234

echo "export to the legacy format (for android etc).."
openssl pkcs12 -export -out $NAME.legacy.p12 -inkey $NAME.key -in $NAME.crt -passin pass:1234 -passout pass:1234



