# Docker volume plugin for Onedata

This plugin allows you to mount Onedata spaces using 'oneclient' directly in Docker containers by creating reusable Docker volumes.

## Usage
This plugin is compatible with both the legacy Docker plugin system (<=1.12) as well as the new managed Docker plugin system (=>1.13).

### Legacy plugin mode

In legacy plugin mode, the plugin should be executed as a host deamon. In this case it is necessary that `oneclient` is properly installed on the host system.

#### Building
Make sure your GOPATH is set properly to your Go development folder.

```
mkdir -p $GOPATH/src/github.com/onedata
cd $GOPATH/src/github.com/onedata
git clone https://github.com/onedata/docker-volume-onedata
cd docker-volume-onedata
make -f Makefile.legacy
sudo make -f Makefile.legacy DESTDIR=/usr/local/bin install
```

#### Running

The Onedata volume plugin should be running as a deamon, it can be invoked manually from command line or configured as a system service.

```
docker-volume-onedata /var/docker/plugins &
```

#### Creating volumes

```
docker volume create -d onedata -o host=<ONEPROVIDER_IP> -o token=<ACCESS_TOKEN> -o insecure=true [-o port=<port>] VOLUME_NAME

docker volume ls
DRIVER              VOLUME NAME
onedata/docker-volume         VOLUME_NAME

docker run -it -v VOLUME_NAME:<path> busybox ls <path>
```


### Managed plugin mode
Since Docker 1.13 plugins are managed by the Docker itself, including publishing and installing them from DockerHub. The plugins are bundled inside of Docker containers.

#### Building
To build the Onedata Docker volume plugin from this repository, execute:
```
make -f Makefile.managed
docker plugin enable onedata/docker-volume:3.0.0-rc12
```

This automatically installs the plugin in the local Docker installation.

#### Installing from DockerHub

To install the plugin directly from DockerHub use:

```
docker plugin install onedata/docker-volume:3.0.0-rc12
docker plugin enable onedata/docker-volume:3.0.0-rc12
```

#### Creating volumes

```
docker volume create -d onedata/docker-volume:3.0.0-rc12 -o host=<ONEPROVIDER_IP> -o token=<ACCESS_TOKEN> -o insecure=true [-o port=<port>] VOLUME_NAME

docker volume ls
DRIVER              VOLUME NAME
onedata/docker-volume         VOLUME_NAME

docker run -it -v VOLUME_NAME:<path> busybox ls <path>
```


## LICENSE
This software is licensed under the Apache 2 license, quoted below.

Licensed under the Apache License, Version 2.0 (the "License"); you may not
use this file except in compliance with the License. You may obtain a copy of
the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
License for the specific language governing permissions and limitations under
the License.