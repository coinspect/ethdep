package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
)

const BASE = "https://api.etherscan.io/api"

type ETHScan struct {
	client *http.Client
	url    string
}

func NewETHScan(apikey string) ETHScan {
	return ETHScan{
		client: http.DefaultClient,
		url:    fmt.Sprintf("%s?apikey=%s", BASE, apikey),
	}
}

type AugmentedSourceCode struct {
	SourceCode         string `json:SourceCode"`
	ConstructArguments []byte
	ContractName       string `json:"ContractName"`
	ABI                []byte
}

func (e *ETHScan) GetSourceCode(addr common.Address) (*AugmentedSourceCode, error) {
	type augmentedSourceCode struct {
		SourceCode         string `json:SourceCode"`
		ConstructArguments string `json:"ConstructorArguments"`
		ContractName       string `json:"ContractName"`
		ABI                string `json:"ABI"`
	}

	type resJSON struct {
		Status  string                `json:"status"`
		Message string                `json:"message"`
		Result  []augmentedSourceCode `json:"result"`
	}

	endpoint := fmt.Sprintf("%s&module=contract&action=getsourcecode&address=%s", e.url, addr)
	res, err := e.client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var j resJSON
	err = json.NewDecoder(res.Body).Decode(&j)
	if err != nil {
		return nil, err
	}
	if j.Message != "OK" {
		return nil, nil
	}
	if len(j.Result) > 1 {
		return nil, errors.New("more than one result?")
	}
	result := j.Result[0]

	if result.ABI == "Contract source code not verified" {
		return nil, errors.New("contract source code not verified")
	}

	constructorArgsBytes, err := hex.DecodeString(result.ConstructArguments)
	if err != nil {
		return nil, err
	}

	return &AugmentedSourceCode{
		SourceCode:         result.SourceCode,
		ConstructArguments: constructorArgsBytes,
		ContractName:       result.ContractName,
		ABI:                []byte(result.ABI),
	}, nil

}

func (e *ETHScan) GetABI(addr common.Address) ([]byte, error) {

	type resJSON struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	endpoint := fmt.Sprintf("%s&module=contract&action=getabi&address=%s", e.url, addr)
	res, err := e.client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var j resJSON
	err = json.NewDecoder(res.Body).Decode(&j)
	if err != nil {
		return nil, err
	}
	if j.Message != "OK" {
		return nil, nil
	}

	return []byte(j.Result), err
}
