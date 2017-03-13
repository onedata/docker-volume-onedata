# Docker volume plugin for Onedata

This plugin allows you to mount Onedata spaces using 'oneclient' directly in Docker containers by creating reusable Docker volumes.

## Usage
This plugin is compatible with both the legacy Docker plugin system as well as the new managed Docker plugin system (not supported yet).

### Legacy plugin mode

In legacy plugin mode, the plugin should be executed as a host deamon. In this case it is necessary that `oneclient` is properly installed on the host system.

#### Building

> Make sure your GOPATH is set properly.

```
mkdir -p $GOPATH/src/github.com/onedata
cd $GOPATH/src/github.com/onedata
git clone https://github.com/onedata/docker-volume-onedata
cd docker-volume-onedata
make -f Makefile.legacy
sudo make -f Makefile.legacy DESTDIR=/usr/local/bin install
```

To build Debian or RPM package use one of the following:
```
make -f Makefile.legacy PROCESS_MANAGER=systemd package-deb
make -f Makefile.legacy PROCESS_MANAGER=systemd package-rpm
make -f Makefile.legacy PROCESS_MANAGER=upstart package-deb
make -f Makefile.legacy PROCESS_MANAGER=upstart package-rpm
```

#### Installing

Onedata `docker-volume-onedata` plugin can be installed using one of the
packages provided for selected distrbutions - see
[get.onedata.org](get.onedata.org).

Please make sure that you have Docker installed.

##### Upstart based distributions (e.g. Ubuntu 14.04 LTS)

After installation `docker-volume-onedata` plugin can be started using `service`
command:

```
sudo service docker-volume-onedata start
```

##### Systemd based distributions

To enable Onedata docker volume to start with the system, use:

```
sudo systemctl enable docker-volume-onedata.service
```

To start:

```
sudo systemctl start docker-volume-onedata.service
```

##### Running manually

The Onedata volume plugin can be started manually as a deamon, it can be invoked manually from command line or configured as a system service.

```
docker-volume-onedata /var/lib/docker/plugins &
```

To run in debug mode use:
```
docker-volume-onedata -d /var/lib/docker/plugins
```

#### Creating volumes

Create the Onedata volume using `docker volume create` commands:

```
docker volume create -d onedata -o host=<ONEPROVIDER_IP> -o token=<ACCESS_TOKEN> -o insecure=true [-o port=<port>] VOLUME_NAME
```

`-o` arguments accept any valid oneclient option, which would be passed
to `oneclient` directly using `--` form, e.g. `-o communicator-thread-count=10`.

Check if the volume was create successfully:
```
docker volume ls
DRIVER              VOLUME NAME
onedata        VOLUME_NAME
```

Show the details of the volume:
```
docker volume inspect plab5
[
    {
        "Driver": "onedata",
        "Labels": {},
        "Mountpoint": "/var/lib/docker/plugins/volumes/6a539918ac2c5baf8c0dbf324fe3826f",
        "Name": "VOLUME_NAME",
        "Options": {
            "host": "oneprovider.example.com",
            "insecure": "true",
            "token": "MDAxNWxvY2F00aW9uIG9uZXpvbmUKMDAzYmlkZW500aWZpZXIgRHR00WTg5dHNHOFZxSzVBZkJhamtaa004wMU5ocWc00azI3WkV00Z00ZkdDJSawowMDFhY2lkIHRpbWUgPCAxNTE5NDgyNDc4CjAwMmZzaWduYXR1cmUgt01Zu6WZ2Wqt3s02nUItRAVDBMYWx6BlBTNQ5KBNqQSDI1"
        },
        "Scope": "local"
    }
]
```

Access volume using any container:
```
docker run -v VOLUME_NAME:/spaces -it busybox ls /spaces
```


### Managed plugin mode

> NOT SUPPORTED YET

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
