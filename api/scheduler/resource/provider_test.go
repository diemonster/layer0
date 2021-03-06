package resource

import (
	"testing"

	"github.com/quintilesims/layer0/common/testutils"
	"github.com/zpatrick/go-bytesize"
)

func TestResourceProviderHasMemoryFor(t *testing.T) {
	cases := []struct {
		Name             string
		ResourceConsumer ResourceConsumer
		Expected         bool
	}{
		{
			Name:             "Port 80 is already used",
			ResourceConsumer: ResourceConsumer{Ports: []int{80}, Memory: bytesize.MB},
			Expected:         false,
		},
		{
			Name:             "Port 8000 is already used",
			ResourceConsumer: ResourceConsumer{Ports: []int{8000}, Memory: bytesize.MB},
			Expected:         false,
		},
		{
			Name:             "Task requires too much memory, no ports",
			ResourceConsumer: ResourceConsumer{Ports: []int{}, Memory: bytesize.GB * 2},
			Expected:         false,
		},
		{
			Name:             "Task requires too much memory, ports are ok",
			ResourceConsumer: ResourceConsumer{Ports: []int{8080}, Memory: bytesize.GB * 2},
			Expected:         false,
		},
		{
			Name:             "Task requires too much memory and already used ports",
			ResourceConsumer: ResourceConsumer{Ports: []int{80, 8000}, Memory: bytesize.GB * 2},
			Expected:         false,
		},
		{
			Name:             "Task requires no resources",
			ResourceConsumer: ResourceConsumer{},
			Expected:         true,
		},
		{
			Name:             "Task requires unused ports",
			ResourceConsumer: ResourceConsumer{Ports: []int{8001, 22, 443}},
			Expected:         true,
		},
		{
			Name:             "Task requires small amounts of available memory",
			ResourceConsumer: ResourceConsumer{Memory: bytesize.MB},
			Expected:         true,
		},
		{
			Name:             "Task requires exact amount of available memory",
			ResourceConsumer: ResourceConsumer{Ports: []int{8080}, Memory: bytesize.GB},
			Expected:         true,
		},
	}

	provider := NewResourceProvider("", true, bytesize.GB, []int{80, 8000})
	for _, c := range cases {
		if output := provider.HasResourcesFor(c.ResourceConsumer); output != c.Expected {
			t.Errorf("%s: output was %t, expected %t", c.Name, output, c.Expected)
		}
	}
}

func TestResourceProviderSubtractResourcesFor(t *testing.T) {
	provider := NewResourceProvider("", false, bytesize.GB, nil)

	resource := ResourceConsumer{Ports: []int{80}}
	if err := provider.SubtractResourcesFor(resource); err != nil {
		t.Error(err)
	}

	resource = ResourceConsumer{Memory: bytesize.MB}
	if err := provider.SubtractResourcesFor(resource); err != nil {
		t.Error(err)
	}

	resource = ResourceConsumer{Ports: []int{8000, 9090}, Memory: bytesize.MB}
	if err := provider.SubtractResourcesFor(resource); err != nil {
		t.Error(err)
	}

	testutils.AssertEqual(t, []int{80, 8000, 9090}, provider.usedPorts)
	testutils.AssertEqual(t, bytesize.GB-(bytesize.MB*2), provider.availableMemory)
	testutils.AssertEqual(t, true, provider.IsInUse())
}

func TestResourceProviderSubtractResourcesForError(t *testing.T) {
	cases := []struct {
		Name             string
		ResourceConsumer ResourceConsumer
	}{
		{
			Name:             "Port 80 already used",
			ResourceConsumer: ResourceConsumer{Ports: []int{80}},
		},
		{
			Name:             "Port 8000 already used",
			ResourceConsumer: ResourceConsumer{Ports: []int{8000}},
		},
		{
			Name:             "Too much memory",
			ResourceConsumer: ResourceConsumer{Memory: bytesize.GB * 2},
		},
	}

	for _, c := range cases {
		provider := NewResourceProvider("", true, bytesize.GB, []int{80, 8000})
		if err := provider.SubtractResourcesFor(c.ResourceConsumer); err == nil {
			t.Fatalf("%s: Error was nil!", c.Name)
		}
	}
}
