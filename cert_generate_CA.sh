
NAME=ca
echo "generate Central Authority key and certificate.."
openssl req -x509 -newkey rsa:4096 -keyout $NAME.key -out $NAME.crt -days 3650

