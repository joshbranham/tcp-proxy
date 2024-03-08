#!/usr/bin/env bash

set -ex

# Generate a root CA cert and private key
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.pem \
  -subj "/C=US/ST=CO/L=Denver/O=Test/CN=root-ca"