//
// @file main.go
// @author Bartek Kryza
// @copyright (C) 2017 ACK CYFRONET AGH
// @copyright This software is released under the MIT license cited in
// 'LICENSE.txt'
//

package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
)

const socketPath = "/run/docker/plugins/onedata.sock"
const oneproviderDefaultPort = "5555"

type OnedataVolume struct {
	// The hostname or IP address of Oneprovider to connect to
	OneproviderHost string
	// The port of Oneprovider (default: 5555)
	OneproviderPort string
	// Users access token to Onedata
	AccessToken string
	// Skip Oneprovider certificate validation (default: false)
	Insecure bool
  // Log directory
  LogDir string
  // Path to config file
  Config string

  // Specify number of parallel buffer scheduler threads
  BufferSchedulerThreadCount string
  // Specify number of parallel communicator threads
  CommunicatorThreadCount string
  // Specify number of parallel scheduler threads
  SchedulerThreadCount string
  // Specify number of parallel storage helper threads
  StorageHelperThreadCount string
  // Specify minimum size in bytes of in-memory cache for input data blocks
  ReadBufferMinSize string
  // Specify maximum size in bytes of in-memory cache for input data blocks
  ReadBufferMaxSize string
  // Specify read ahead period in seconds of in-memory cache for input data
  // blocks
  ReadBufferPrefetchDuration string
  // Specify minimum size in bytes of in-memory cache for output data blocks
  WriteBufferMinSize string
  // Specify maximum size in bytes of in-memory cache for output data blocks
  WriteBufferMaxSize string
  // Specify idle period in seconds before flush of in-memory cache for output
  // data blocks
  WriteBufferFlushDelay string
	// Fuse options
	FuseOptions string

	// Mountpoint on host where the Oneclient will mount the Fuse filesystem
	Mountpoint string

	// Number of containers connected to this volume
	connections int
}

type OnedataDriver struct {
	sync.RWMutex

	root      string
	statePath string
	volumes   map[string]*OnedataVolume
}

func newOnedataDriver(root string) (*OnedataDriver, error) {
	log.WithField("method", "new driver").Debug(root)

	d := &OnedataDriver{
		root:      filepath.Join(root, "volumes"),
		statePath: filepath.Join(root, "onedata-state.json"),
		volumes:   map[string]*OnedataVolume{},
	}

	data, err := ioutil.ReadFile(d.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.WithField("statePath", d.statePath).Debug("no state found")
		} else {
			return nil, err
		}
	} else {
		if err := json.Unmarshal(data, &d.volumes); err != nil {
			return nil, err
		}
	}

	return d, nil
}

func (d *OnedataDriver) saveState() {
	data, err := json.Marshal(d.volumes)
	if err != nil {
		log.WithField("statePath", d.statePath).Error(err)
		return
	}

	if err := ioutil.WriteFile(d.statePath, data, 0644); err != nil {
		log.WithField("savestate", d.statePath).Error(err)
	}
}

func (d *OnedataDriver) Create(r volume.Request) volume.Response {
	log.WithField("method", "create").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()
	v := &OnedataVolume{}

	// Set default values
	v.Insecure = false
	v.OneproviderPort = oneproviderDefaultPort

	for key, val := range r.Options {
		switch key {
		case "host":
			v.OneproviderHost = val
		case "token":
			v.AccessToken = val
		case "port":
			v.OneproviderPort = val
		case "insecure":
			if strings.EqualFold(val, "true") {
				v.Insecure = true
			}
    case "opt":
      options := append(strings.Split(v.FuseOptions, ","),
                        strings.Split(val, ",")...)
      v.FuseOptions = strings.Join(options, ",")
    case "buffer-scheduler-thread-count":
      v.BufferSchedulerThreadCount = val
    case "communicator-thread-count":
      v.CommunicatorThreadCount = val
    case "scheduler-thread-count":
      v.SchedulerThreadCount = val
    case "storage-helper-thread-count":
      v.StorageHelperThreadCount = val
    case "read-buffer-min-size":
      v.ReadBufferMinSize = val
    case "read-buffer-max-size":
      v.ReadBufferMaxSize = val
    case "read-buffer-prefetch-duration":
      v.ReadBufferPrefetchDuration = val
    case "write-buffer-min-size":
      v.WriteBufferMinSize = val
    case "write-buffer-max-size":
      v.WriteBufferMaxSize = val
    case "write-buffer-flush-delay":
      v.WriteBufferFlushDelay = val
		default:
			return responseError(fmt.Sprintf("Unknown option %q", val))
		}
	}

	if v.OneproviderHost == "" {
		return responseError(
			"Oneprovider host must be specified using 'host=' option!")
	}
	if v.AccessToken == "" {
		return responseError(
			"Access token must be specified using 'token=' option!")
	}

	// Generate unique path for the mountpoint based on Oneprovider host
	// and access token
	v.Mountpoint = filepath.Join(d.root, fmt.Sprintf("%x",
		md5.Sum([]byte(v.OneproviderHost+v.AccessToken))))

	d.volumes[r.Name] = v

	d.saveState()

	return volume.Response{}
}

func (d *OnedataDriver) Remove(r volume.Request) volume.Response {
	log.WithField("method", "remove").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("Volume %s not found", r.Name))
	}

	if v.connections != 0 {
		return responseError(fmt.Sprintf(
			"Volume %s is currently used by a container", r.Name))
	}

	if err := os.RemoveAll(v.Mountpoint); err != nil {
		return responseError(err.Error())
	}

	delete(d.volumes, r.Name)
	d.saveState()

	return volume.Response{}
}

func (d *OnedataDriver) Path(r volume.Request) volume.Response {
	log.WithField("method", "path").Debugf("%#v", r)

	d.RLock()
	defer d.RUnlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("volume %s not found", r.Name))
	}

	return volume.Response{Mountpoint: v.Mountpoint}
}

func (d *OnedataDriver) Mount(r volume.MountRequest) volume.Response {
	log.WithField("method", "mount").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("Volume %s not found", r.Name))
	}

	if v.connections == 0 {
		fi, err := os.Lstat(v.Mountpoint)
		if os.IsNotExist(err) {
			if err := os.MkdirAll(v.Mountpoint, 0755); err != nil {
				return responseError(err.Error())
			}
		} else if err != nil {
			return responseError(err.Error())
		}

		if fi != nil && !fi.IsDir() {
			return responseError(
				fmt.Sprintf("%v already exist and it's not a directory", v.Mountpoint))
		}

		if err := d.mountVolume(v); err != nil {
			return responseError(err.Error())
		}
	}

	v.connections++

	return volume.Response{Mountpoint: v.Mountpoint}
}

func (d *OnedataDriver) Unmount(r volume.UnmountRequest) volume.Response {
	log.WithField("method", "unmount").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()
	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("Volume %s not found", r.Name))
	}

	v.connections--

	if v.connections <= 0 {
		if err := d.unmountVolume(v.Mountpoint); err != nil {
			return responseError(err.Error())
		}
		v.connections = 0
	}

	return volume.Response{}
}

func (d *OnedataDriver) Get(r volume.Request) volume.Response {
	log.WithField("method", "get").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	v, ok := d.volumes[r.Name]
	if !ok {
		return responseError(fmt.Sprintf("Volume %s not found", r.Name))
	}

	return volume.Response{Volume: &volume.Volume{Name: r.Name,
		Mountpoint: v.Mountpoint}}
}

func (d *OnedataDriver) List(r volume.Request) volume.Response {
	log.WithField("method", "list").Debugf("%#v", r)

	d.Lock()
	defer d.Unlock()

	var vols []*volume.Volume
	for name, v := range d.volumes {
		vols = append(vols, &volume.Volume{Name: name, Mountpoint: v.Mountpoint})
	}
	return volume.Response{Volumes: vols}
}

func (d *OnedataDriver) Capabilities(r volume.Request) volume.Response {
	log.WithField("method", "capabilities").Debugf("%#v", r)

	return volume.Response{Capabilities: volume.Capability{Scope: "local"}}
}

func (d *OnedataDriver) mountVolume(v *OnedataVolume) error {
	//
	// Build oneclient mount command from OnedataVolume parameters
	//
	cmd := fmt.Sprintf("oneclient -H %s -t %s", v.OneproviderHost, v.AccessToken)

	//
	// Add optional arguments
	//
	if v.OneproviderPort != oneproviderDefaultPort {
		cmd = fmt.Sprintf("%s -P %s", cmd, v.OneproviderPort)
	}
	if v.Insecure {
		cmd = fmt.Sprintf("%s -i", cmd)
	}
	if v.FuseOptions != "" {
		cmd = fmt.Sprintf("%s --opt %s", cmd, v.FuseOptions)
	}
  if v.BufferSchedulerThreadCount != "" {
		cmd = fmt.Sprintf("%s --buffer-scheduler-thread-count %s", cmd,
      v.BufferSchedulerThreadCount)
	}
  if v.CommunicatorThreadCount != "" {
		cmd = fmt.Sprintf("%s --communicator-thread-count %s", cmd,
      v.CommunicatorThreadCount)
	}
  if v.SchedulerThreadCount != "" {
		cmd = fmt.Sprintf("%s --scheduler-thread-count %s", cmd,
      v.SchedulerThreadCount)
	}
  if v.StorageHelperThreadCount != "" {
		cmd = fmt.Sprintf("%s --storage-helper-thread-count %s", cmd,
      v.StorageHelperThreadCount)
	}
  if v.ReadBufferMinSize != "" {
		cmd = fmt.Sprintf("%s --read-buffer-min-size %s", cmd,
      v.ReadBufferMinSize)
	}
  if v.ReadBufferMaxSize != "" {
		cmd = fmt.Sprintf("%s --read-buffer-max-size %s", cmd,
      v.ReadBufferMaxSize)
	}
  if v.ReadBufferPrefetchDuration != "" {
		cmd = fmt.Sprintf("%s --read-buffer-prefetch-duration %s", cmd,
      v.ReadBufferPrefetchDuration)
	}
  if v.WriteBufferMinSize != "" {
		cmd = fmt.Sprintf("%s --write-buffer-min-size %s", cmd,
      v.WriteBufferMinSize)
	}
  if v.WriteBufferMaxSize != "" {
		cmd = fmt.Sprintf("%s --write-buffer-max-size %s", cmd,
      v.WriteBufferMaxSize)
	}
  if v.WriteBufferFlushDelay != "" {
		cmd = fmt.Sprintf("%s --write-buffer-flush-delay %s", cmd,
      v.WriteBufferFlushDelay)
	}

	//
	// Add Docker plugin mountpoint
	//
	cmd = fmt.Sprintf("%s %s", cmd, v.Mountpoint)

	log.Debug(cmd)

	return exec.Command("sh", "-c", cmd).Run()
}

func (d *OnedataDriver) unmountVolume(target string) error {
	cmd := fmt.Sprintf("oneclient -u %s", target)
	log.Debug(cmd)
	return exec.Command("sh", "-c", cmd).Run()
}

func responseError(err string) volume.Response {
	log.Error(err)
	return volume.Response{Err: err}
}

func printUsage(progName string) {
	fmt.Println("Onedata Docker volume plugin")
	fmt.Println("Usage:")
	fmt.Println("\t", progName, "[-d] [-h] <docker_plugins_path>")
}

func main() {

	commandLineArguments := os.Args
	programName := commandLineArguments[0]

	if len(commandLineArguments) < 2 {
		log.Fatal("Too few arguments to %s", programName)
		printUsage(programName)
		os.Exit(1)
	}

	//
	// If second argument is -h print help
	//
	if commandLineArguments[1] == "-h" {
		printUsage(programName)
		os.Exit(0)
	}

	//
	// If second argument is -d enable debug mode
	//
	if commandLineArguments[1] == "-d" {
		log.SetLevel(log.DebugLevel)
	}

	//
	// The last parameter should be a path to the Docker plugins directory
	//
	pluginsRoot := commandLineArguments[len(commandLineArguments)-1]
	fmt.Println("Plugins root:", pluginsRoot)
	fileInfo, err := os.Stat(pluginsRoot)
	if err != nil || !fileInfo.IsDir() {
		fmt.Println("Invalid path to Docker plugins root, try: /run/docker/plugins")
		os.Exit(1)
	}

	d, err := newOnedataDriver(pluginsRoot)
	if err != nil {
		log.Fatal(err)
	}

	h := volume.NewHandler(d)
	if runtime.GOOS == "linux" {
		log.Infof("Listening on Unix socket: %s", socketPath)
		log.Error(h.ServeUnix(socketPath, 0))
	} else {
		log.Error("This operating system is not supported: ", runtime.GOOS)
	}

}
