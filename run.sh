#!/bin/sh

git pull origin do;docker up -d --force-recreate --build fetch;docker system prune -a;