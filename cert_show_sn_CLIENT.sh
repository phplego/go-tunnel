NAME=client

openssl x509 -in $NAME.crt -noout -text
openssl x509 -in $NAME.crt -noout -serial
