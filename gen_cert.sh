#!/bin/sh

#openssl req -new -key cert.key -subj "/CN=$1" -sha256 | openssl x509 -req -days 3650 -CA ca.crt -CAkey ca.key -set_serial "$2" 
openssl req -new -key cert.key -subj "/CN=$1" -addext "subjectAltName=DNS:$1" -sha256 | \
openssl x509 -req -days 3650 -CA ca.crt -CAkey ca.key -set_serial "$2" -extfile <(echo "subjectAltName=DNS:$1")