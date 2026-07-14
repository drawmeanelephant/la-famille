#!/bin/bash
go fmt ./...
if [[ -n $(git status -s) ]]; then
  echo "Go files need formatting."
fi
