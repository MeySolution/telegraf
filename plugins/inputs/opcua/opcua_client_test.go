package opcua_client

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/influxdata/telegraf/config"
	"github.com/stretchr/testify/require"
)

type OPCTags struct {
	Name           string
	Namespace      string
	IdentifierType string
	Identifier     string
	Want           string
}

func TestClient1(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	var testopctags = []OPCTags{
		{"ProductName", "0", "i", "2261", "open62541 OPC UA Server"},
		{"ProductUri", "0", "i", "2262", "http://open62541.org"},
		{"ManufacturerName", "0", "i", "2263", "open62541"},
	}

	var o OpcUA
	var err error

	o.MetricName = "testing"
	o.Endpoint = "opc.tcp://opcua.rocks:4840"
	o.AuthMethod = "Anonymous"
	o.ConnectTimeout = config.Duration(10 * time.Second)
	o.RequestTimeout = config.Duration(1 * time.Second)
	o.SecurityPolicy = "None"
	o.SecurityMode = "None"
	for _, tags := range testopctags {
		o.RootNodes = append(o.RootNodes, MapOPCTag(tags))
	}
	err = o.Init()
	if err != nil {
		t.Errorf("Initialize Error: %s", err)
	}
	err = Connect(&o)
	if err != nil {
		t.Fatalf("Connect Error: %s", err)
	}

	for i, v := range o.nodeData {
		if v.Value != nil {
			types := reflect.TypeOf(v.Value)
			value := reflect.ValueOf(v.Value)
			compare := fmt.Sprintf("%v", value.Interface())
			if compare != testopctags[i].Want {
				t.Errorf("Tag %s: Values %v for type %s  does not match record", o.nodes[i].tag.FieldName, value.Interface(), types)
			}
		} else {
			t.Errorf("Tag: %s has value: %v", o.nodes[i].tag.FieldName, v.Value)
		}
	}
}

func MapOPCTag(tags OPCTags) (out NodeSettings) {
	out.FieldName = tags.Name
	out.Namespace = tags.Namespace
	out.IdentifierType = tags.IdentifierType
	out.Identifier = tags.Identifier
	return out
}

func TestConfig(t *testing.T) {
	toml := `
[[inputs.opcua]]
metric_name = "localhost"
endpoint = "opc.tcp://localhost:4840"
connect_timeout = "10s"
request_timeout = "5s"
security_policy = "auto"
security_mode = "auto"
certificate = "/etc/telegraf/cert.pem"
private_key = "/etc/telegraf/key.pem"
auth_method = "Anonymous"
username = ""
password = ""
nodes = [
  {field_name="name", namespace="1", identifier_type="s", identifier="one"},
  {field_name="name2", namespace="2", identifier_type="s", identifier="two"},
]
[[inputs.opcua.group]]
metric_name = "foo"
namespace = "3"
identifier_type = "i"
nodes = [{field_name="name3", identifier="3000"}]
`

	c := config.NewConfig()
	err := c.LoadConfigData([]byte(toml))
	require.NoError(t, err)

	require.Len(t, c.Inputs, 1)

	o, ok := c.Inputs[0].Input.(*OpcUA)
	require.True(t, ok)

	require.Len(t, o.RootNodes, 2)
	require.Equal(t, o.RootNodes[0].FieldName, "name")
	require.Equal(t, o.RootNodes[1].FieldName, "name2")

	require.Len(t, o.Groups, 1)
	require.Equal(t, o.Groups[0].MetricName, "foo")
	require.Len(t, o.Groups[0].Nodes, 1)
	require.Equal(t, o.Groups[0].Nodes[0].Identifier, "3000")

	require.NoError(t, o.InitNodes())
	require.Len(t, o.nodes, 3)
}
