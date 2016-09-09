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
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/andygrunwald/megos"
	"github.com/golang/glog"
)

var (
	mfile = flag.String("file.master", "mesos-master.json", "File to write mesos-master info to.")
	sfile = flag.String("file.slaves", "mesos-slaves.json", "File to write mesos-slaves info to.")

	cvsrPort = flag.Int("cadvisor.port", 4194, "Port for cadvisor on the nodes, 0 to disable")
	ndxpPort = flag.Int("nodeexporter.port", 9100, "Port for node_exporter on the nodes, 0 to disable")
)

var cn string

func main() {
	flag.Parse()

	ms := []*url.URL{}
	for _, murl := range flag.Args() {
		n, _ := url.Parse(murl)
		ms = append(ms, n)
	}

	wg := &sync.WaitGroup{}
	ctx := context.Background()

	// Cluster name is only on the master, we'll grab it now
	// to pass to slave workers
	mesos := megos.NewClient(ms)
	leader, err := mesos.DetermineLeader()
	if err != nil {
		log.Fatalf("Can't get initial cluster leader, ", err.Error())
	}

	_, state, err := addrFromPID(mesos, leader.String())
	if err != nil {
		log.Fatalf("Can't get initial cluster leader, ", err.Error())
	}

	cn = state.Cluster

	wg.Add(2)
	mw := make(chan []target)
	go configUpdater(ctx, *mfile, "mesos-master-exprter", mw, wg)
	go masterWatcher(ctx, mw, wg, ms)

	wg.Add(2)
	sw := make(chan []target)
	go configUpdater(ctx, *sfile, "mesos-slave-exporter", sw, wg)
	go slaveWatcher(ctx, sw, wg, ms)

	wg.Wait()
}

type target struct {
	pid      string
	remAddr  string
	remPort  string
	remState *megos.State
	remAttrs map[string]interface{}
}

type byURL []target

func (s byURL) Len() int           { return len(s) }
func (s byURL) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byURL) Less(i, j int) bool { return s[i].pid < s[j].pid }

func configUpdater(ctx context.Context, fn, jobn string, targets chan []target, wg *sync.WaitGroup) {
	oldtargets := []target{}
	for {
		select {
		case <-ctx.Done():
		case newts := <-targets:
			sort.Sort(byURL(newts))
			same := true
			if len(oldtargets) != len(newts) {
				same = false
			} else {
				for i := range oldtargets {
					if oldtargets[i].remAddr != newts[i].remAddr {
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

func writeConfig(fn, jobn string, targets []target) {
	type promTargetGroup struct {
		Targets []string          `json:"targets"`
		Labels  map[string]string `json:"labels",omitempty`
	}

	f, err := os.Create(fn)
	if err != nil {
		glog.Errorf("Error updating %s, %s", fn, err.Error())
		return
	}

	tgs := []promTargetGroup{}
	tgsHostDone := map[string]bool{}
	for _, t := range targets {
		// Add the specific mesos service target
		attrs := map[string]string{
			"job":                jobn,
			"mesos_cluster_name": cn,
		}
		for k, v := range t.remAttrs {
			if s, ok := v.(string); ok {
				attrs["mesos_"+strings.ToLower(k)] = s
			}
		}
		tgs = append(tgs,
			promTargetGroup{
				[]string{t.remState.Hostname + ":" + t.remPort},
				attrs,
			})

		if _, ok := tgsHostDone[t.remState.Hostname]; ok {
			// already added target for this host
			continue
		}

		// Add cadvisor and node_exporter
		delete(attrs, "instance")

		if *cvsrPort != 0 {
			cvsrAttrs := map[string]string{}
			for k, v := range attrs {
				cvsrAttrs[k] = v
			}
			cvsrAttrs["job"] = "cadvisor"
			tgs = append(tgs,
				promTargetGroup{
					[]string{fmt.Sprintf("%s:%d", t.remState.Hostname, *cvsrPort)},
					cvsrAttrs,
				},
			)
		}
		if *ndxpPort != 0 {
			nodeAttrs := map[string]string{}
			for k, v := range attrs {
				nodeAttrs[k] = v
			}
			nodeAttrs["job"] = "node"
			tgs = append(tgs,
				promTargetGroup{
					[]string{fmt.Sprintf("%s:%d", t.remState.Hostname, *ndxpPort)},
					nodeAttrs,
				},
			)
		}

		tgsHostDone[t.remState.Hostname] = true
	}

	enc := json.NewEncoder(f)
	enc.Encode(tgs)
	f.Close()
}

func addrFromPID(mesos *megos.Client, PID string) (string, *megos.State, error) {
	var pid *megos.Pid
	var err error
	if pid, err = mesos.ParsePidInformation(PID); err != nil {
		return "", nil, err
	}

	state, err := mesos.GetStateFromPid(pid)
	if err != nil {
		return "", nil, fmt.Errorf("Could not get state, %s", err.Error())
	}

	return fmt.Sprintf("http://%s:%d", pid.Host, pid.Port), state, nil
}

func slaveWatcher(ctx context.Context, writer chan []target, wg *sync.WaitGroup, ms []*url.URL) {
	glog.Info("Running Slave Watcher")

	pidmap := map[string]bool{}
	mesos := megos.NewClient(ms)
	for {
		select {
		case <-time.After(time.Second * 5):
			cstate, err := mesos.GetSlavesFromCluster()
			if err != nil {
				glog.Error("Could not get list of slaves, ", err.Error())
				continue
			}
			newss := cstate.Slaves

			targets := []target{}
			// Kill old watchers
			for oldpid, _ := range pidmap {
				var found bool
				for _, news := range newss {
					if news.PID == oldpid && news.Active {
						addr := ""

						addr, state, err := addrFromPID(mesos, news.PID)
						if err != nil {
							glog.Errorf("Could not parse slave state, %s ", err.Error())
							continue
						}

						targets = append(targets, target{pid: news.PID, remAddr: addr, remPort: "9110", remState: state, remAttrs: news.Attributes})
						found = true
						break
					}
				}
				if !found {
					glog.Infof("Slave %s has left\n", oldpid)
					delete(pidmap, oldpid)
				}
			}

			// Start new watchers
			for _, news := range newss {
				var err error
				if _, ok := pidmap[news.PID]; !ok && news.Active {
					pidmap[news.PID] = true

					var addr string
					var state *megos.State
					if addr, state, err = addrFromPID(mesos, news.PID); err != nil {
						glog.Errorf("Could not parse pid %s, %s\n", news.PID, err.Error())
						continue
					}
					targets = append(targets, target{pid: news.PID, remAddr: addr, remPort: "9110", remState: state, remAttrs: news.Attributes})
				}
			}
			writer <- targets
		}
	}
}

func masterWatcher(ctx context.Context, writer chan []target, wg *sync.WaitGroup, ms []*url.URL) {
	glog.Info("Running Master Watcher")

	mesos := megos.NewClient(ms)
	oldleaderpid := ""

	for {
		select {
		case <-time.After(time.Second * 5):
			leader, err := mesos.DetermineLeader()
			addr, state, err := addrFromPID(mesos, leader.String())
			if err != nil {
				glog.Error("Could not parse leader PID %#v, %s ", leader, err.Error())
				continue
			}

			if leader.String() != oldleaderpid {
				oldleaderpid = leader.String()

				writer <- []target{{
					remAddr:  addr,
					remPort:  "9105",
					remState: state,
				}}
			}
		}
	}
	wg.Done()
}
