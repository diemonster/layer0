package command

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/quintilesims/layer0/cli/printer"
	"github.com/quintilesims/layer0/cli/resolver/mock_resolver"
	"github.com/quintilesims/layer0/client/mock_client"
)

type TestCommandBase struct {
	Client   *mock_client.MockClient
	Printer  *printer.TestPrinter
	Resolver *mock_resolver.MockResolver
}

func newTestCommand(t *testing.T) (*TestCommandBase, *gomock.Controller) {
	ctrl := gomock.NewController(t)

	tc := &TestCommandBase{
		Client:   mock_client.NewMockClient(ctrl),
		Printer:  &printer.TestPrinter{},
		Resolver: mock_resolver.NewMockResolver(ctrl),
	}

	return tc, ctrl
}

func (c *TestCommandBase) Command() *CommandBase {
	return &CommandBase{
		client:   c.Client,
		printer:  c.Printer,
		resolver: c.Resolver,
	}
}