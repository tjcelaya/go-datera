package dsdk_test

import (
	"dsdk"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const (
	ADDR     = "172.19.1.41"
	PORT     = "7717"
	APIVER   = "2.1"
	USERNAME = "admin"
	PASSWORD = "password"
	TENANT   = "/root"
	TIMEOUT  = "30s"
	TOKEN    = "test1234"
)

// TODO (_alastor_) implement real unit tests using these mocked structures
// currently all the following tests are "integration tests" since they require
// being pointed at a working cluster via the above constants
type mockHTTPClient struct {
}

func (c *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {

	return &http.Response{}, nil
}

type mockAPIConnection struct {
	Method     string
	Endpoint   string
	Headers    map[string]string
	QParams    []string
	Hostname   string
	APIVersion string
	Port       string
	Secure     bool
	Client     *mockHTTPClient
	Tenant     string
	Auth       *dsdk.LogAuth
}

func (r mockAPIConnection) UpdateHeaders(h ...string) error {
	fmt.Println("Headers:", h)
	return nil
}

func (r mockAPIConnection) Login() error {
	fmt.Println("Login")
	r.Auth.SetToken(TOKEN)
	return nil
}

func (r mockAPIConnection) Get(endpoint string, qparams ...string) ([]byte, error) {
	fmt.Println(endpoint, qparams)
	return []byte(""), nil
}

func (r mockAPIConnection) Put(endpoint string, sensitive bool, bodyp ...interface{}) ([]byte, error) {
	fmt.Println(endpoint, sensitive, bodyp)
	return []byte(""), nil
}

func (r mockAPIConnection) Post(endpoint string, bodyp ...interface{}) ([]byte, error) {
	fmt.Println(endpoint, bodyp)
	return []byte(""), nil
}

func (r mockAPIConnection) Delete(endpoint string, bodyp ...interface{}) ([]byte, error) {
	fmt.Println(endpoint, bodyp)
	return []byte(""), nil
}

func getClient(t *testing.T) *dsdk.Client {
	headers := make(map[string]string)
	client, err := dsdk.NewClient(
		ADDR, PORT, USERNAME, PASSWORD, APIVER, TENANT, TIMEOUT, headers, false)
	if err != nil {
		t.Fatalf("%s", err)
	}
	// Mock the connection pool clients
	// auth := dsdk.NewAuth("test", "pass")
	// for i := 0; i <= dsdk.MaxPoolConn; i++ {
	// 	<-dsdk.Cpool.Conns
	// 	dsdk.Cpool.Conns <- &mockAPIConnection{Auth: auth}
	// }
	return client
}

func TestApiBasic(t *testing.T) {

}

func TestConnection(t *testing.T) {
	headers := make(map[string]string)
	auth := dsdk.NewLogAuth("admin", "password")
	conn, err := dsdk.NewAPIConnection("172.19.1.41", "7717", "2.1", "/root", "30s", headers, false, auth)
	if err != nil {
		t.Fatalf("%s", err)
	}
	conn.UpdateHeaders("Content-Type=application/json")
	err = conn.Login()
	if err != nil {
		t.Fatalf("%s", err)
	}
	_, err = conn.Get("users")
	if err != nil {
		t.Fatalf("%s", err)
	}

}

func TestEndpoint(t *testing.T) {
	client := getClient(t)
	_, err := client.GetEp("app_instances").List()
	if err != nil {
		t.Fatalf("%s", err)
	}
}

func TestSubendpoint(t *testing.T) {
	client := getClient(t)
	name, _ := dsdk.NewUUID()
	ai, err := client.GetEp("app_instances").Create(
		fmt.Sprintf("name=%s", name))
	ai.GetEp("storage_instances").Create()
	ais, err := client.GetEp("app_instances").List()
	if err != nil {
		t.Fatalf("%s", err)
	}
	ai = ais[0]
	ai.GetEp("storage_instances").Create("name=storage-1")
	ai, _ = ai.Reload()
	si := ai.GetEn("storage_instances")[0]
	si, err = si.Reload()
	if err != nil {
		t.Fatalf("%s", err)
	}
}

func TestCreate(t *testing.T) {
	client := getClient(t)
	name, _ := dsdk.NewUUID()
	ai, err := client.GetEp("app_instances").Create(
		fmt.Sprintf("name=%s", name))
	if err != nil {
		t.Fatalf("%s", err)
	}
	ai, err = ai.Reload()
	if err != nil {
		t.Fatalf("%s", err)
	}
	ai, err = ai.Set("admin_state=offline")
	if err != nil {
		t.Fatalf("%s", err)
	}
	err = ai.Delete("force=true")
	if err != nil {
		t.Fatalf("%s", err)
	}
}

func TestACL(t *testing.T) {
	client := getClient(t)
	name, _ := dsdk.NewUUID()
	ai, err := client.GetEp("app_instances").Create(
		fmt.Sprintf("name=%s", name))
	if err != nil {
		t.Fatalf("%s", err)
	}
	si, _ := ai.GetEp("storage_instances").Create("name=storage-1")
	initep := client.GetEp("initiators")
	_, err = initep.Create(
		"name=test-initiator",
		"id=iqn.1993-08.org.debian:01:71be38c985a")
	if err != nil {
		t.Fatalf("%s", err)
	}
	aclep := si.GetEp("acl_policy")
	var args map[string]interface{}
	err = json.Unmarshal([]byte(`{"initiators":[{"path": "/initiators/iqn.1993-08.org.debian:01:71be38c985a"}]}`), &args)
	if err != nil {
		t.Fatalf("%s", err)
	}
	_, err = aclep.Set(args)
	if err != nil {
		t.Fatalf("%s", err)
	}
}

func TestConcurrency(t *testing.T) {
	client := getClient(t)
	n := dsdk.MaxPoolConn * 5
	var dones []chan int
	for i := 0; i <= n; i++ {
		dones = append(dones, make(chan int))
	}
	f := func(lc chan int) {
		client.GetEp("app_instances").List()
		lc <- 1
	}
	for _, c := range dones {
		go f(c)
	}
	for _, c := range dones {
		<-c
	}

}

func TestAutoGenEntities(t *testing.T) {
	client := getClient(t)
	name, _ := dsdk.NewUUID()
	siname := "storage-1"
	ai, _ := client.GetEp("app_instances").Create(
		fmt.Sprintf("name=%s", name))
	ai.GetEp("storage_instances").Create(
		fmt.Sprintf("name=%s", siname))

	ai, err := ai.Reload()

	var enai dsdk.AppInstance
	err = json.Unmarshal(ai.GetB(), &enai)
	if err != nil {
		t.Fatalf("%s", err)
	}
	if (*enai.StorageInstances)[0].Name != siname {
		t.Fatalf(
			"Storage Instance name doesn't match.  Expected: %s, Actual %s",
			siname, (*enai.StorageInstances)[0].Name)
	}
}

func TestReadme(t *testing.T) {
	client := getClient(t)
	// Now that we have the client, lets create an AppInstance
	// Each call to a SubEndpoint is done via the "GetEp" function
	ai, err := client.GetEp("app_instances").Create("name=my-app")
	if err != nil {
		panic(err)
	}

	// This call returns a genric Entity Object.  The attributes of this
	// object can be accessed in two ways

	// 1. The dynamic way via the original JSON key
	aiName := ai.Get("name").(string)
	fmt.Printf("Dynamic Name: %s\n", aiName)

	// 2. The static way via unpacking into an autogenerated object
	var myai dsdk.AppInstance
	err = json.Unmarshal(ai.GetB(), &myai)
	fmt.Printf("Static Name: %s\n", myai.Name)

	// Now lets update that AppInstance
	// You can pass two types of arguments to Create/Set/Delete functions

	// 1. "key=value" strings, both arguments MUST be strings when this form is used
	ai.Set("descr=my test label")
	ai, _ = ai.Reload()
	json.Unmarshal(ai.GetB(), &myai)
	fmt.Printf("Description: %s\n", myai.Descr)

	// 2. Give a single struct or map[string]interface{}
	var sendAi dsdk.AppInstance
	sendAi.Descr = "golden ticket"
	ai.Set(sendAi)
	ai, _ = ai.Reload()
	json.Unmarshal(ai.GetB(), &myai)
	fmt.Printf("Description2: %s\n", myai.Descr)

	// Just for fun, lets create an AppInstance, StorageInstance and Volume
	// Then online and print the connection info
	testVol := dsdk.Volume{
		Name:         "my-vol",
		Size:         5,
		ReplicaCount: 1,
	}
	testSi := dsdk.StorageInstance{
		Name:    "my-si",
		Volumes: &[]dsdk.Volume{testVol},
	}
	testAi := dsdk.AppInstance{
		Name:             "my-ai",
		StorageInstances: &[]dsdk.StorageInstance{testSi},
	}
	ai, err = client.GetEp("app_instances").Create(testAi)
	ai, err = ai.Reload()
	var myAi dsdk.AppInstance
	if err != nil {
		t.Fatalf("%s", err)
	}
	err = json.Unmarshal(ai.GetB(), &myAi)
	if err != nil {
		t.Fatalf("%s", err)
	}
	mySi := (*myAi.StorageInstances)[0]
	myVol := (*mySi.Volumes)[0]
	fmt.Printf("AI Path: %s\nSI Path: %s\nVol Path: %s\n", myAi.Path, mySi.Path, myVol.Path)

	// Get the storage_instance endpoint, send "admin_state=online" and update our struct
	sis, _ := ai.GetEp("storage_instances").List()
	si := sis[0]
	si.Set("admin_state=online")
	si, _ = si.Reload()
	json.Unmarshal(si.GetB(), &mySi)
	fmt.Printf("Access: %s", mySi.Access.(map[string]interface{}))
}

func TestClean(t *testing.T) {
	client := getClient(t)
	var dones []chan int
	f := func(lc chan int, en dsdk.IEntity) {
		if strings.Contains(en.GetPath(), "app_instances") {
			en.Set("admin_state=offline", "force=true")
		}
		en.Delete("force=true")
		lc <- 1
	}

	// Count number of requests we need to make
	ais, _ := client.GetEp("app_instances").List()
	inits, _ := client.GetEp("initiators").List()

	// Populate channel array
	ldones := len(ais) + len(inits)
	for i := 0; i < ldones; i++ {
		dones = append(dones, make(chan int))
	}

	// Initiate goroutines with channels
	chi := 0
	for _, ai := range ais {
		go f(dones[chi], ai)
		chi += 1
	}
	for _, init := range inits {
		go f(dones[chi], init)
		chi += 1
	}

	// Check channels for completion
	for _, ch := range dones {
		<-ch
	}
}
