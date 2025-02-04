#!/bin/sh

git pull origin do;docker compose up -d --force-recreate --build fetch;docker system prune -af;
