# ethdep

An utility to find linked addresses in a contract in the Ethereum network.

The linked addresses are found by:

1. Directly inspecting the [EIP1967](https://eips.ethereum.org/EIPS/eip-1967) slots to find the implementation, beacon and admin addresses. The storage is inspected onchain via a JSON-RPC provider.

2. Asking [Etherscan](https://etherscan.io/) for the contract source code and trying to find any getters which reffers to addresses. If the source code is verified, we can also retrieve it's `name` for displaying it nicely in the terminal.

This process is recursive and it stops at `maxDepth` recursive calls. So you can get linked addresses of linked address of a contract, and so forth.


This tool is highly **experimental** and very rough around the edges. Using it should pose absolutely no risk, but you will most probably find bugs and problems. Feel free to hack around!

## Example usage


``` sh
$ go run . -addr="0xEe6A57eC80ea46401049E92587E52f5Ec1c24785" -ethscankey=$ETHERSCAN_API -jsonrpc=$INFURA_MAINNET -maxDepth=3

- Target contract: 0xEe6A57eC80ea46401049E92587E52f5Ec1c24785
- Max depth: 3
- Starting. This may take a while...

0xEe6A57eC80ea46401049E92587E52f5Ec1c24785
--> (EIP1967 ADMIN - ProxyAdmin): 0xB753548F6E010e7e680BA186F9Ca1BdAB2E90cf2
----> (owner - ): 0x6C9FC64A53c1b71FB3f9Af64d1ae3A4931A5f4E9
--> (EIP1967 BEACON - ): 0x0000000000000000000000000000000000000000
--> (EIP1967 IMPL - NonfungibleTokenPositionDescriptor): 0x91ae842A5Ffd8d12023116943e72A606179294f3
----> (WETH9 - WETH9): 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2
--> (admin - ): 0x0000000000000000000000000000000000000000
--> (implementation - ): 0x0000000000000000000000000000000000000000
```

As you probably guessed, you need to set valid values for `-ethscankey` and `-jsonrpc` as you wish. `-maxDepth 3` is usually reasonable, but adjust as needed.
