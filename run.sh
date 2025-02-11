#!/bin/sh
cd /root/app
sudo docker compose stop



sudo docker system prune -af
sudo docker volume prune -f
sudo docker image prune -f
sudo docker container prune -f
sudo docker network prune -f

rm -rf /var/snap/docker/common/var-lib-docker/overlay2/*
sudo snap stop docker 
sudo snap remove docker --purge




git add .;
git commit -m "[refactor] grafana";
git pull origin do;
git push origin do;


sudo snap install docker
snap restart docker
sudo compose up -d --force-recreate --build
sudo docker system prune -af
sudo docker volume prune -f
sudo docker image prune -f
sudo docker container prune -f
sudo docker network prune -f
