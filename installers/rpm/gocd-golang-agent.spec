Name:           gocd-golang-agent
Version:        0.1
Release:        1
Summary:        GoCD Golang Agent
License:        Apache License Version 2.0
URL:            https://github.com/gocd-contrib/gocd-golang-agent
SOURCE:         gocd-golang-agent-0.1.tar.gz

%description
GoCD Golang Agent

%prep
%setup

%install
rm -rf "$RPM_BUILD_ROOT"
mkdir -p "$RPM_BUILD_ROOT/opt/%{name}/bin"
mkdir -p "$RPM_BUILD_ROOT/etc/init.d"
mkdir -p "$RPM_BUILD_ROOT/etc/default"
mkdir -p "$RPM_BUILD_ROOT/var/log/%{name}"
mkdir -p "$RPM_BUILD_ROOT/var/lib/%{name}"
cp opt/%{name}/bin/agent.sh "$RPM_BUILD_ROOT/opt/%{name}/bin"
cp opt/%{name}/bin/gocd-golang-agent "$RPM_BUILD_ROOT/opt/%{name}/bin"
cp etc/init.d/gocd-golang-agent "$RPM_BUILD_ROOT/etc/init.d"
cp etc/default/gocd-golang-agent "$RPM_BUILD_ROOT/etc/default"

%pre
getent group go >/dev/null || groupadd go
getent passwd go >/dev/null || \
    useradd -g go -d /var/go -s /bin/bash \
    -c "GoCD Golang Agent" go
mkdir -p /var/lib/%{name}
mkdir -p /var/log/%{name}

%preun
service %{name} stop

%postun
rm -rf /var/log/%{name}
rm -rf /var/lib/%{name}

%files
/opt/%{name}
/etc/init.d/gocd-golang-agent
/etc/default/gocd-golang-agent

%attr(755, root, root) /etc/init.d/gocd-golang-agent
%attr(644, go, go) /etc/default/gocd-golang-agent
%attr(755, go, go) /opt/%{name}/bin/gocd-golang-agent
%attr(755, go, go) /opt/%{name}/bin/agent.sh
%attr(755, go, go) /var/log/%{name}
%attr(755, go, go) /var/lib/%{name}

%clean
rm -rf "$RPM_BUILD_ROOT"

%changelog
* Fri May 27 2016  Barrow Kwan <bhkwan@thoughtworks.com> 1.0-1
- First Build
