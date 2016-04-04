GoLang agent for GoCD
=========================

GOCD agent golang implementation. Comparing to java implementation, golang agent has less installation dependency, less memory footprint and shorter boostrap time. More suitable for running in container.

Golang agent is based on "BuildCommand API" proposed [here](https://github.com/gocd/gocd/issues/1954). We are working on contributing serverside implementation to GOCD codebase. Meanwhile you can run golang agent against server in this experimental GOCD fork: https://github.com/wpc/gocd/tree/build_command_protocol.

### Installation

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

### Configure Agent

Agent is designed to be configured by environment variables. The followings are available options:

* **GOCD_SERVER_URL**: Go server url, default to https://localhost:8154/go.
* **GOCD_AGENT_WORKING_DIR**: Agent working directory, default to Agent script launch directory. All build data will be inside this directory.
* **GOCD_AGENT_CONFIG_DIR**: Agent configurations for connecting to Go server, default to be "config" directory inside **GOCD_AGENT_WORKING_DIR** directory
* **GOCD_AGENT_LOG_DIR**: Agent log directory, without this configuration, log will be output to stdout.
* **DEBUG**: set this environment variable to any value will turn on debug log.

## Contributing

Bug reports and pull requests are welcome on GitHub at https://github.com/gocd-contrib/gocd-golang-agent.

[Document for Developer](/gocd-contrib/gocd-golang-agent/wiki/For-Developer)

