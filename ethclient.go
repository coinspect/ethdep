package main

import (
	"context"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ETHClient struct {
	client *ethclient.Client
}

func NewETHClient(endpoint string) (*ETHClient, error) {
	c, err := ethclient.Dial(endpoint)
	return &ETHClient{client: c}, err
}

func (c *ETHClient) CallContractGetAddr(atAddr common.Address, fnSelector []byte) common.Address {
	ret := c.CallContract(atAddr, fnSelector)
	return common.BytesToAddress(ret)

}

func (c *ETHClient) CallContract(atAddr common.Address, fnSelector []byte) []byte {
	ctx := context.Background()
	res, err := c.client.CallContract(ctx, ethereum.CallMsg{
		To:   &atAddr,
		Data: fnSelector,
	}, nil)
	if err != nil {
		// execution reverted is expected, some times the getter
		// is behind authentication
		if err.Error() != "execution reverted" {
			panic(err)
		}
	}
	return res

}

func (c *ETHClient) GetStorage(atAddr common.Address, keyBytes []byte) []byte {
	ctx := context.Background()
	key := common.BytesToHash(keyBytes)
	res, err := c.client.StorageAt(ctx, atAddr, key, nil)
	if err != nil {
		panic(err)
	}
	return res

}

func (c *ETHClient) GetEIP1967Slots(contract common.Address) (EIP1967Slots, error) {
	ctx := context.Background()
	slots := &EIP1967Slots{}

	implSlot := common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")
	beaconSlot := common.HexToHash("0xa3f0ad74e5423aebfd80d3ef4346578335a9a72aeaee59ff6cb3582b35133d50")
	adminSlot := common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103")

	implAddr, err := c.client.StorageAt(ctx, contract, implSlot, nil)
	if err != nil {
		return *slots, err
	}
	slots.ImplementationAddr = common.BytesToAddress(implAddr)

	beaconAddr, err := c.client.StorageAt(ctx, contract, beaconSlot, nil)
	if err != nil {
		return *slots, err
	}
	slots.BeaconAddr = common.BytesToAddress(beaconAddr)
	adminAddr, err := c.client.StorageAt(ctx, contract, adminSlot, nil)
	slots.AdminAddr = common.BytesToAddress(adminAddr)

	return *slots, err

}
