GoLang agent for GoCD
=========================

GOCD agent golang implementation. Comparing to java implementation, golang agent has less installation dependency, less memory footprint and shorter boostrap time. More suitable for running in container.

Golang agent is based on "BuildCommand API" proposed [here](https://github.com/gocd/gocd/issues/1954). We are working on contributing serverside implementation to GOCD codebase. Meanwhile you can run golang agent against server in this experimental GOCD fork: https://github.com/wpc/gocd/tree/build_command_protocol.

Maturity
=======
Experimental

Installation
===========

On Ubuntu:
* Add Bintray's GPG key:
```
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys 379CE192D401AB61
```
* Add repo
```
sudo echo deb https://dl.bintray.com/alex-hal9000/gocd-golang-agent master main | sudo tee -a /etc/apt/sources.list
```
* Install
```
sudo apt-get install gocd-golang-agent
```
