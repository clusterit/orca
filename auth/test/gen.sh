#!/bin/sh

openssl genrsa -out app.rsa
openssl rsa -in app.rsa -pubout > app.rsa.pub
