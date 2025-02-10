#!/bin/sh
cd /root/app;
git add .;
git commit -m "[refactor] grafana";
git pull origin do;
git push origin do;
docker compose up -d --force-recreate --build fetch;
docker system prune -af;
