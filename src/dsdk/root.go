package dsdk

import (
	"context"
	"strings"
	"time"
)

const (
	VERSION = "1.0.7"
)

var (
	Cpool *connectionPool
)

type SDK struct {
}

func NewPoolSDK(hostname, username, password, apiVersion, tenant, timeout string, headers map[string]string, secure bool, logfile string, stdout bool) (*SDK, error) {
	var err error
	//Initialize global connection object
	Cpool, err = newConnPool(hostname, username, password, apiVersion, tenant, timeout, headers, secure, logfile, stdout, false)
	if err != nil {
		return nil, err
	}
	conn := Cpool.getConn()
	defer Cpool.releaseConn(conn)
	return &SDK{}, nil
}

func NewSDK(hostname, username, password, apiVersion, tenant, timeout string, headers map[string]string, secure bool, logfile string, stdout bool) (*SDK, error) {
	var err error
	Cpool, err = newConnPool(hostname, username, password, apiVersion, tenant, timeout, headers, secure, logfile, stdout, true)
	if err != nil {
		return nil, err
	}
	conn := Cpool.getConn()
	defer Cpool.releaseConn(conn)
	return &SDK{}, nil
}

func (c SDK) GetEp(path string) IEndpoint {
	return newEp("", path)
}

// Cleans AppInstances, AppTemplates, StorageInstances, Initiators and InitiatorGroups under
// the currently configured tenant
func (c SDK) ForceClean() {
	ctxt := context.TODO()
	f := func(lc chan int, en IEntity) {
		if strings.Contains(en.GetPath(), "app_instances") {
			en.Set(ctxt, "admin_state=offline", "force=true")
		}
		if strings.Contains(en.GetPath(), "app_templates") {
			for {
				err := en.Delete(ctxt, "force=true")
				if err != nil {
					if strings.Contains(err.Error(), "read-only") {
						break
					} else {
						time.Sleep(2 * time.Second)
					}
				} else {
					break
				}
			}
		}
		en.Delete(ctxt, "force=true")
		lc <- 1
	}

	var dones []chan int
	chi := 0
	for _, epStr := range []string{"app_instances", "app_templates", "initiators", "initiator_groups"} {
		items, _ := c.GetEp(epStr).List(ctxt)
		numItems := len(items)
		for i := 0; i < numItems; i++ {
			dones = append(dones, make(chan int))
		}
		for _, item := range items {
			go f(dones[chi], item)
			chi += 1
		}
	}
	// Check channels for completion
	for _, ch := range dones {
		<-ch
	}
}
