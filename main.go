package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type Contract struct {
	Addr            common.Address
	GivenName       string
	OwnName         string
	ParentContract  *Contract
	AdminContract   *Contract
	ImplContract    *Contract
	BeaconContract  *Contract
	LinkedContracts []*Contract
}

func (c *Contract) AddLinkedAddress(name string, addr common.Address) {
	child := &Contract{
		ParentContract: c,
		Addr:           addr,
		GivenName:      name,
	}
	c.LinkedContracts = append(c.LinkedContracts, child)

}

func (c *Contract) AddEIP1967Children(children EIP1967Slots) {

	if c.AdminContract != nil ||
		c.BeaconContract != nil ||
		c.ImplContract != nil {
		panic("trying to add eip1967 children again!")
	}

	c.AdminContract = &Contract{
		Addr:           children.AdminAddr,
		ParentContract: c,
		GivenName:      "EIP1967 ADMIN",
	}

	c.ImplContract = &Contract{
		Addr:           children.ImplementationAddr,
		ParentContract: c,
		GivenName:      "EIP1967 IMPL",
	}
	c.BeaconContract = &Contract{
		Addr:           children.BeaconAddr,
		ParentContract: c,
		GivenName:      "EIP1967 BEACON",
	}
}

func (c *Contract) AddDependencies(eth *ETHClient, ethscan *ETHScan, depth, maxDepth int) {
	if depth == maxDepth {
		return
	}

	augmentedSourceCode, err := ethscan.GetSourceCode(c.Addr)
	if err != nil && err.Error() != "contract source code not verified" {
		panic(err)
	}

	// XXX: probably the worst...
	setOfLinked := make(map[string]struct{})
	// XXX: rate limit gotten or the agumented source code was not verified
	if augmentedSourceCode != nil {
		c.OwnName = augmentedSourceCode.ContractName

		parsedABI, err := ParseABI(augmentedSourceCode.ABI)
		if err != nil {
			panic(err)
		}

		selectors := AddressGettersToSelectors(parsedABI.Methods)
		for name, sel := range selectors {
			res := eth.CallContractGetAddr(c.Addr, sel)
			c.AddLinkedAddress(name, res)
			setOfLinked[res.Hex()] = struct{}{}
		}
	} else {
		// default to only abi...
		abi, err := ethscan.GetABI(c.Addr)
		if err != nil {
			panic(err)
		}
		if abi != nil {
			parsedABI, err := ParseABI(abi)
			if err != nil {
				panic(err)
			}
			c.OwnName = "???"

			selectors := AddressGettersToSelectors(parsedABI.Methods)
			for name, sel := range selectors {
				res := eth.CallContractGetAddr(c.Addr, sel)
				c.AddLinkedAddress(name, res)
				setOfLinked[res.Hex()] = struct{}{}
			}

		}

	}

	slots, err := eth.GetEIP1967Slots(c.Addr)
	if err != nil {
		panic(err)
	}
	if !slots.Empty() {
		c.AddEIP1967Children(slots)
		c.AdminContract.AddDependencies(eth, ethscan, depth+1, maxDepth)
		c.BeaconContract.AddDependencies(eth, ethscan, depth+1, maxDepth)
		c.ImplContract.AddDependencies(eth, ethscan, depth+1, maxDepth)
	}
	if len(c.LinkedContracts) > 0 {
		for _, linkedContract := range c.LinkedContracts {
			linkedContract.AddDependencies(eth, ethscan, depth+1, maxDepth)
		}
	}

}

func (c *Contract) buildString(depth int) string {
	prefix := strings.Repeat("--", depth)
	s := fmt.Sprintf("%s\n", c.Addr)
	if c.AdminContract != nil {
		s += fmt.Sprintf("--%s> (%s - %s): %s", prefix, c.AdminContract.GivenName, c.AdminContract.OwnName, c.AdminContract.buildString(depth+1))
	}
	if c.BeaconContract != nil {
		s += fmt.Sprintf("--%s> (%s - %s): %s", prefix, c.BeaconContract.GivenName, c.BeaconContract.OwnName, c.BeaconContract.buildString(depth+1))
	}
	if c.ImplContract != nil {
		s += fmt.Sprintf("--%s> (%s - %s): %s", prefix, c.ImplContract.GivenName, c.ImplContract.OwnName, c.ImplContract.buildString(depth+1))
	}
	for _, linkedAddr := range c.LinkedContracts {
		s += fmt.Sprintf("--%s> (%s - %s): %s", prefix, linkedAddr.GivenName, linkedAddr.OwnName, linkedAddr.buildString(depth+1))
	}
	return s

}

func (c *Contract) String() string {
	return c.buildString(0)
}

type Flags struct {
	TargetAddr        string
	ETHScanAPIKey     string
	ETHClientEndpoint string
	MaxDepth          int
}

func parseFlags() Flags {
	addr := flag.String("addr", "", "the address of the target contract, 0x prefixed")
	ethscanApiKey := flag.String("ethscankey", "", "an Etherscan API Key")
	ethclientEndpoint := flag.String("jsonrpc", "", "an Ethereum JSON RPC endpoint")
	maxDepth := flag.Int("maxDepth", 5, "the max recursive depth")
	flag.Parse()

	if *addr == "" {
		log.Fatal("[ERR] target address not set")
	}
	if *ethscanApiKey == "" {
		log.Fatal("[ERR] no Etherscan API key set")
	}
	if *ethclientEndpoint == "" {
		log.Fatal("[ERR] no JSON-RPC endpoint set")
	}

	return Flags{*addr, *ethscanApiKey, *ethclientEndpoint, *maxDepth}
}

func main() {
	flags := parseFlags()
	// XXX: api keys!!!
	eth, err := NewETHClient(flags.ETHClientEndpoint)
	if err != nil {
		log.Fatalf("[ERR] there was a problem initializing the JSON-RPC client: %s", err)
	}
	ethscan := NewETHScan(flags.ETHScanAPIKey)
	targetContract := common.HexToAddress(flags.TargetAddr)

	fmt.Printf("- Target contract: %s\n", flags.TargetAddr)
	fmt.Printf("- Max depth: %d\n", flags.MaxDepth)
	fmt.Printf("- Starting. This may take a while...\n")

	contract := Contract{Addr: targetContract}
	contract.AddDependencies(eth, &ethscan, 0, flags.MaxDepth)
	fmt.Printf("\n%s", contract.String())
}

type EIP1967Slots struct {
	ImplementationAddr common.Address
	BeaconAddr         common.Address
	AdminAddr          common.Address
}

func (e EIP1967Slots) Empty() bool {
	return e.ImplementationAddr == common.Address{} &&
		e.BeaconAddr == common.Address{} &&
		e.AdminAddr == common.Address{}

}

// XXX: TODO
// CONTINUE PARSING THE SOURCE CODE
type SourceCode struct {
	StateVariable bool `json:"stateVariable"`
}

func ParseSourceCode(sourceCode string) error {
	tmp, err := ioutil.TempFile("/tmp", "ethdep")
	if err != nil {
		return err
	}
	_, err = tmp.WriteString(sourceCode)
	if err != nil {
		return err
	}

	cmd := exec.Command("solc", "--ast-compact-json", tmp.Name())
	outputRaw, err := cmd.Output()
	if err != nil {
		return err
	}

	var output string
	for i, c := range outputRaw {
		if c == '{' {
			output = string(outputRaw[i:])
		}
	}
	output += "foo"

	return nil

}

func ParseABI(rawABI []byte) (abi.ABI, error) {
	return abi.JSON(strings.NewReader(string(rawABI)))
}

func AddressGettersToSelectors(methods map[string]abi.Method) map[string][]byte {
	selectors := make(map[string][]byte)

	isRelevantMethod := func(m abi.Method) bool {
		return len(m.Outputs) == 1 && // getters return only one thing
			m.Type == abi.Function && // getters are not fallback, constructors, etc
			len(m.Inputs) == 0 && // getters have no inputs
			m.Outputs[0].Type.String() == "address" // and we only care about addr
	}

	for _, m := range methods {
		if isRelevantMethod(m) {
			selectors[m.Name] = m.ID
		}
	}
	return selectors
}
