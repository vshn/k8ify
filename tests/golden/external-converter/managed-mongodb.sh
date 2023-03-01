#!/bin/bash
echo "apiVersion: mongodb.appcat.vshn.io/v1
kind: ManagedMongoDB
metadata:
  name: $1
spec:
  parameters:
    backup:
      timeOfDay: \"02:00:00\" 
    service:
      zone: ch-dk-2 
      majorVersion: \"4\" 
      mongoSettings:
        timezone: Europe/Zurich 
    size:
      plan: $3
  writeConnectionSecretToRef:
    name: mongo-creds"
