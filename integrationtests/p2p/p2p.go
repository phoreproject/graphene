package testcase

import (
	"fmt"

	"github.com/multiformats/go-multiaddr"

	"github.com/phoreproject/synapse/integrationtests/framework"
)

// P2pTest implements IntegrationTest
type P2pTest struct {
}

// Execute implements IntegrationTest
func (test P2pTest) Execute(service *testframework.TestService) error {
	arg := service.GetArgString("arg", "n/a")
	fmt.Printf("This is sample test, arg = " + arg)
	return nil
}

func (test P2pTest) createNodeAddress(index int) *multiaddr.Multiaddr {
	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/202.0.5.1/tcp/%d", 9000+index))
	return &addr
}
