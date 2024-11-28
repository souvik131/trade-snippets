#!/bin/sh


sudo go build -o build/linux;
sudo service fetch stop;
sudo systemctl daemon-reload;
sudo service fetch start;
