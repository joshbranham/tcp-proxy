#!/usr/bin/env bash

set -ex

# Create a private key and CSR for the proxy, then sign
openssl genrsa -out tcp-proxy.key 2048
openssl req -new -key tcp-proxy.key -out tcp-proxy.csr \
  -subj "/C=US/ST=CO/L=Denver/O=Test/CN=tcp-proxy"

openssl x509 -req -in tcp-proxy.csr -CA ca.pem -CAkey ca.key \
  -out tcp-proxy.pem -days 365 -sha256 \
  -extfile san_config.ext

# Create a private key and CSR for user1 in group engineering
openssl genrsa -out user1.key 2048
openssl req -new -key user1.key -out user1.csr \
  -subj "/C=US/ST=CO/L=Denver/O=Test/CN=user1@engineering"

# Create a signed certificate from CSR for user1
openssl x509 -req -in user1.csr -CA ca.pem -CAkey ca.key \
  -out user1.pem -days 365 -sha256 \
  -extfile san_config.ext

# Create a private key and CSR for user2 in group administrators
openssl genrsa -out user2.key 2048
openssl req -new -key user2.key -out user2.csr \
  -subj "/C=US/ST=CO/L=Denver/O=Test/CN=user2@administrators"

# Create a signed certificate from CSR for user2
openssl x509 -req -in user2.csr -CA ca.pem -CAkey ca.key \
  -out user2.pem -days 365 -sha256 \
  -extfile san_config.ext