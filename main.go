package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/guelfey/go.dbus"
	"github.com/marpaia/graphite-golang"
)

type jobBody struct {
	JobID    uint32
	JobPath  dbus.ObjectPath
	UnitName string
	Status   string
}

type graphiteEvent struct {
	What string   `json:"what"`
	Tags []string `json:"tags"`
	Data string   `json:"data"`
}

func main() {
	conn, err := dbus.SystemBus()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to session bus:", err)
		os.Exit(1)
	}

	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0,
		"type='signal',path='/org/freedesktop/systemd1',interface='org.freedesktop.systemd1.Manager'")

	c := make(chan *dbus.Signal, 10)
	conn.Signal(c)
	for s := range c {

		if s.Name == "org.freedesktop.systemd1.Manager.JobNew" ||
			s.Name == "org.freedesktop.systemd1.Manager.JobRemoved" {

			jobNameArr := strings.Split(s.Name, ".")
			job := &jobBody{
				JobID:    s.Body[0].(uint32),
				JobPath:  s.Body[1].(dbus.ObjectPath),
				UnitName: s.Body[2].(string),
			}
			if len(s.Body) == 4 {
				job.Status = s.Body[3].(string)
			}

			g := &graphite.Graphite{}
			configGraphite := false
			if configGraphite {
				g, _ = graphite.NewGraphite("localhost", 2003)
			} else {
				g = graphite.NewGraphiteNop("localhost", 2003)
			}

			metric := fmt.Sprintf("systemd.%s.%s.%s",
				jobNameArr[len(jobNameArr)-1],
				strings.Replace("%{hostname}", ".", "_", -1),
				strings.Replace(job.UnitName, ".", "_", -1),
			)
			g.SimpleSend(metric, "1")
		}
	}
}
