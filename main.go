// Copyright 2016 Qubit Digital Ltd  All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/andygrunwald/megos"
	"github.com/golang/glog"
)

var (
	ecmd  = flag.String("mesos-exporter", "mesos-exporter", "Binary for the mesos-exporter.")
	mfile = flag.String("file.master", "mesos-master.json", "File to write mesos-master info to.")
	sfile = flag.String("file.slaves", "mesos-slaves.json", "File to write mesos-slaves info to.")
	mport = flag.Int("mport", 10000, "Port for the master exporter to run on")
	sbase = flag.Int("sbase", 10001, "Port for the slave exporters to run on, all ports above this port will be attempted")
)

func main() {
	flag.Parse()

	ms := []*url.URL{}
	for _, murl := range flag.Args() {
		n, _ := url.Parse(murl)
		ms = append(ms, n)
	}

	wg := &sync.WaitGroup{}
	ctx := context.Background()

	wg.Add(2)
	mw := make(chan []string)
	go configUpdater(ctx, *mfile, "mesos-master", mw, wg)
	go masterWatcher(ctx, mw, wg, ms)

	wg.Add(2)
	sw := make(chan []string)
	go configUpdater(ctx, *sfile, "mesos-slave", sw, wg)
	go slaveWatcher(ctx, sw, wg, ms)

	wg.Wait()
}

func configUpdater(ctx context.Context, fn, jobn string, targets chan []string, wg *sync.WaitGroup) {
	oldtargets := []string{}
	for {
		select {
		case <-ctx.Done():
		case newts := <-targets:
			sort.Strings(newts)
			same := true
			if len(oldtargets) != len(newts) {
				same = false
			} else {
				for i := range oldtargets {
					if oldtargets[i] != newts[i] {
						same = false
					}
				}
			}
			if same {
				continue
			}

			writeConfig(fn, jobn, newts)
			oldtargets = newts
		}
	}
	wg.Done()
}

func writeConfig(fn, jobn string, targets []string) {
	type promTargetGroup struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels",omitempty`
	}

	f, err := os.Create(fn)
	if err != nil {
		glog.Errorf("Error updating %s, %s", fn, err.Error())
		return
	}

	enc := json.NewEncoder(f)
	enc.Encode([]promTargetGroup{{
		targets,
		map[string]string{"job": jobn},
	}})
	f.Close()
}

func addrFromPID(mesos *megos.Client, p string) (string, error) {
	if pid, err := mesos.ParsePidInformation(p); err != nil {
		return "", err
	} else {
		return fmt.Sprintf("http://%s:%d", pid.Host, pid.Port), nil
	}
}

func slaveWatcher(ctx context.Context, writer chan []string, wg *sync.WaitGroup, ms []*url.URL) {
	glog.Info("Running Slave Watcher")
	type sslot struct {
		port int
		cf   context.CancelFunc
	}
	smap := map[string]*sslot{}
	highport := *sbase - 1

	spool := &sync.Pool{
		New: func() interface{} {
			highport++
			return &sslot{port: highport}
		},
	}

	mesos := megos.NewClient(ms)
	for {
		select {
		case <-ctx.Done():
			// Stop any running exporters
			for _, s := range smap {
				if s.cf != nil {
					s.cf()
				}
			}
		case <-time.After(time.Second * 5):
			state, err := mesos.GetSlavesFromCluster()
			if err != nil {
				glog.Error("Could not get list of slaves, ", err.Error())
				continue
			}
			newss := state.Slaves

			targets := []string{}
			// Kill old watchers
			for oldpid, slot := range smap {
				var found bool
				for _, news := range newss {
					if news.PID == oldpid {
						expaddr := fmt.Sprintf("localhost:%d", slot.port)
						targets = append(targets, expaddr)
						found = true
						break
					}
				}
				if !found {
					glog.Infof("Slave %s has left\n", oldpid)
					slot.cf()
					slot.cf = nil
					spool.Put(slot)
				}
			}

			// Start new watchers
			for _, news := range newss {
				var addr string
				var err error
				if _, ok := smap[news.PID]; !ok {
					if slot, ok := spool.Get().(*sslot); !ok || slot == nil {
						glog.Error("Could not get slave slot")
					} else {
						sctx, scf := context.WithCancel(ctx)
						slot.cf = scf
						smap[news.PID] = slot

						if addr, err = addrFromPID(mesos, news.PID); err != nil {
							glog.Errorf("Could no parse pid %s\n", news.PID)
							continue
						}
						expaddr := fmt.Sprintf("localhost:%d", slot.port)
						targets = append(targets, expaddr)
						args := []string{"-addr", expaddr, "-slave", addr}
						go processWatcher(sctx, *ecmd, args...)
					}
				}
			}

			writer <- targets
		}
	}
	wg.Done()
}

func masterWatcher(ctx context.Context, writer chan []string, wg *sync.WaitGroup, ms []*url.URL) {
	glog.Info("Running Master Watcher")

	mesos := megos.NewClient(ms)
	oldleaderpid := ""
	var mcf context.CancelFunc
	var mctx context.Context

	for {

		select {
		case <-ctx.Done():
			// Stop any running exporters
			if mcf != nil {
				mcf()
				mcf = nil
			}
		case <-time.After(time.Second * 5):
			leader, err := mesos.DetermineLeader()
			if err != nil {
				glog.Error("Could not get leader, ", err.Error())
				continue
			}

			addr, err := addrFromPID(mesos, leader.String())
			if err != nil {
				glog.Error("Could not parse leader PID %#v, %s ", leader, err.Error())
				continue
			}

			if leader.String() != oldleaderpid {
				if mcf != nil {
					mcf()
					mcf = nil
				}
				mctx, mcf = context.WithCancel(ctx)

				expaddr := fmt.Sprintf("localhost:%d", *mport)
				args := []string{"-addr", expaddr, "-master", addr}
				go processWatcher(mctx, *ecmd, args...)

				oldleaderpid = leader.String()

				writer <- []string{expaddr}
			}
		}
	}
	wg.Done()
}

// Loops restarting a process
func processWatcher(ctx context.Context, name string, args ...string) {
	for {
		glog.Infof("Running %s %s\n", *ecmd, strings.Join(args, " "))
		fin := make(chan struct{})

		cmd := exec.Command(name, args...)
		go func() {
			if out, err := cmd.CombinedOutput(); err != nil {
				glog.Error("Failed to run ", string(out), err)
			}
			close(fin)
		}()
		select {
		case <-fin:
		case <-ctx.Done():
			cmd.Process.Kill()
			return
		}
	}
}
