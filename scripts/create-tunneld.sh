#!/usr/bin/env bash


curl -X POST -d @scripts/tunneld-service.json "http://localhost:6400/services/create"