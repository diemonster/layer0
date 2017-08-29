package system

import (
	"strconv"
	"testing"

	"github.com/quintilesims/layer0/common/config"
	"github.com/quintilesims/layer0/tests/clients"
	"github.com/quintilesims/tftest"
)

const (
	serviceDeployCommand = "while true ; do echo LONG RUNNING SERVICE ; sleep 5 ; done"
	taskDeployCommand    = "echo SHORT RUNNING TASK ; sleep 10"
)

type StressTestCase struct {
	NumDeploys        int
	NumDeployFamilies int
	DeployCommand     string
	NumEnvironments   int
	NumLoadBalancers  int
	NumServices       int
}

func runTest(b *testing.B, c StressTestCase) {
	vars := map[string]string{
		"endpoint":            config.APIEndpoint(),
		"token":               config.AuthToken(),
		"num_deploys":         strconv.Itoa(c.NumDeploys),
		"num_deploy_families": strconv.Itoa(c.NumDeployFamilies),
		"deploy_command":      c.DeployCommand,
		"num_environments":    strconv.Itoa(c.NumEnvironments),
		"num_load_balancers":  strconv.Itoa(c.NumLoadBalancers),
		"num_services":        strconv.Itoa(c.NumServices),
	}

	tfContext := tftest.NewTestContext(
		b,
		tftest.Dir("module"),
		tftest.Vars(vars),
		tftest.DryRun(*dry),
		tftest.Log(b),
	)

	layer0 := clients.NewLayer0TestClient(b, vars["endpoint"], vars["token"])

	tfContext.Apply()
	defer tfContext.Destroy()

	methodsToBenchmark := map[string]func(){
		"ListEnvironments":  func() { layer0.ListEnvironments() },
		"ListLoadBalancers": func() { layer0.ListLoadBalancers() },
		"ListDeploys":       func() { layer0.ListDeploys() },
		"ListServices":      func() { layer0.ListServices() },
		"ListTasks":         func() { layer0.ListTasks() },
	}

	benchmark(b, methodsToBenchmark)
}

func benchmark(b *testing.B, methods map[string]func()) {
	for name, fn := range methods {
		b.Run(name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				fn()
			}
		})
	}
}

func Benchmark5Services(b *testing.B) {
	runTest(b, StressTestCase{
		NumEnvironments: 2,
		NumServices:     5,
		NumDeploys:      1,
		DeployCommand:   serviceDeployCommand,
	})
}
