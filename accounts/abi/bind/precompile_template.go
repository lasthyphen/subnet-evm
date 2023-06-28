// (c) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package bind

// tmplPrecompileData is the data structure required to fill the binding template.
type tmplPrecompileData struct {
	Contract *tmplPrecompileContract // The contract to generate into this file
	Structs  map[string]*tmplStruct  // Contract struct type definitions
}

// tmplPrecompileContract contains the data needed to generate an individual contract binding.
type tmplPrecompileContract struct {
	*tmplContract
	AllowList bool                   // Indicator whether the contract uses AllowList precompile
	Funcs     map[string]*tmplMethod // Contract functions that include both Calls + Transacts in tmplContract
}

// tmplSourcePrecompileGo is the Go precompiled source template.
const tmplSourcePrecompileGo = `
// Code generated
// This file is a generated precompile contract with stubbed abstract functions.
// The file is generated by a template. Please inspect every code and comment in this file before use.

// There are some must-be-done changes waiting in the file. Each area requiring you to add your code is marked with CUSTOM CODE to make them easy to find and modify.
// Additionally there are other files you need to edit to activate your precompile.
// These areas are highlighted with comments "ADD YOUR PRECOMPILE HERE".
// For testing take a look at other precompile tests in core/stateful_precompile_test.go

/* General guidelines for precompile development:
1- Read the comment and set a suitable contract address in precompile/params.go. E.g:
	{{.Contract.Type}}Address = common.HexToAddress("ASUITABLEHEXADDRESS")
2- Set gas costs here
3- It is recommended to only modify code in the highlighted areas marked with "CUSTOM CODE STARTS HERE". Modifying code outside of these areas should be done with caution and with a deep understanding of how these changes may impact the EVM.
Typically, custom codes are required in only those areas.
4- Add your upgradable config in params/precompile_config.go
5- Add your precompile upgrade in params/config.go
6- Add your solidity interface and test contract to contract-examples/contracts
7- Write solidity tests for your precompile in contract-examples/test
8- Create your genesis with your precompile enabled in tests/e2e/genesis/
9- Create e2e test for your solidity test in tests/e2e/solidity/suites.go
10- Run your e2e precompile Solidity tests with './scripts/run_ginkgo.sh'

*/

package precompile

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/lasthyphen/subnet-evm/accounts/abi"
	"github.com/lasthyphen/subnet-evm/vmerrs"

	"github.com/ethereum/go-ethereum/common"
)

const (
	{{- range .Contract.Funcs}}
	{{.Normalized.Name}}GasCost uint64 = 0 // SET A GAS COST HERE
	{{- end}}
	{{- if .Contract.Fallback}}
	{{.Contract.Type}}FallbackGasCost uint64 = 0 // SET A GAS COST LESS THAN 2300 HERE
  {{- end}}

	// {{.Contract.Type}}RawABI contains the raw ABI of {{.Contract.Type}} contract.
	{{.Contract.Type}}RawABI = "{{.Contract.InputABI}}"
)

// CUSTOM CODE STARTS HERE
// Reference imports to suppress errors from unused imports. This code and any unnecessary imports can be removed.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = fmt.Printf
)

{{$contract := .Contract}}
// Singleton StatefulPrecompiledContract and signatures.
var (
	_ StatefulPrecompileConfig = &{{.Contract.Type}}Config{}

	{{- range .Contract.Funcs}}

	{{- if not .Original.IsConstant | and $contract.AllowList}}

	ErrCannot{{.Normalized.Name}} = errors.New("non-enabled cannot call {{.Original.Name}}")
	{{- end}}
	{{- end}}

	{{- if .Contract.Fallback | and $contract.AllowList}}
	Err{{.Contract.Type}}CannotFallback = errors.New("non-enabled cannot call fallback function")
	{{- end}}

	{{.Contract.Type}}ABI abi.ABI // will be initialized by init function

	{{.Contract.Type}}Precompile StatefulPrecompiledContract // will be initialized by init function

	// CUSTOM CODE STARTS HERE
	// THIS SHOULD BE MOVED TO precompile/params.go with a suitable hex address.
	{{.Contract.Type}}Address = common.HexToAddress("ASUITABLEHEXADDRESS")
)

// {{.Contract.Type}}Config implements the StatefulPrecompileConfig
// interface while adding in the {{.Contract.Type}} specific precompile address.
type {{.Contract.Type}}Config struct {
	{{- if .Contract.AllowList}}
	AllowListConfig
	{{- end}}
	UpgradeableConfig
}

{{$structs := .Structs}}
{{range $structs}}
	// {{.Name}} is an auto generated low-level Go binding around an user-defined struct.
	type {{.Name}} struct {
	{{range $field := .Fields}}
	{{$field.Name}} {{$field.Type}}{{end}}
	}
{{- end}}

{{- range .Contract.Funcs}}
{{ if len .Normalized.Inputs | lt 1}}
type {{capitalise .Normalized.Name}}Input struct{
{{range .Normalized.Inputs}} {{capitalise .Name}} {{bindtype .Type $structs}}; {{end}}
}
{{- end}}
{{ if len .Normalized.Outputs | lt 1}}
type {{capitalise .Normalized.Name}}Output struct{
{{range .Normalized.Outputs}} {{capitalise .Name}} {{bindtype .Type $structs}}; {{end}}
}
{{- end}}
{{- end}}

func init() {
	parsed, err := abi.JSON(strings.NewReader({{.Contract.Type}}RawABI))
	if err != nil {
		panic(err)
	}
	{{.Contract.Type}}ABI = parsed

	{{.Contract.Type}}Precompile = create{{.Contract.Type}}Precompile({{.Contract.Type}}Address)
}

// New{{.Contract.Type}}Config returns a config for a network upgrade at [blockTimestamp] that enables
// {{.Contract.Type}} {{if .Contract.AllowList}} with the given [admins] as members of the allowlist {{end}}.
func New{{.Contract.Type}}Config(blockTimestamp *big.Int{{if .Contract.AllowList}}, admins []common.Address{{end}}) *{{.Contract.Type}}Config {
	return &{{.Contract.Type}}Config{
		{{if .Contract.AllowList}}AllowListConfig:   AllowListConfig{AllowListAdmins: admins},{{end}}
		UpgradeableConfig: UpgradeableConfig{BlockTimestamp: blockTimestamp},
	}
}

// NewDisable{{.Contract.Type}}Config returns config for a network upgrade at [blockTimestamp]
// that disables {{.Contract.Type}}.
func NewDisable{{.Contract.Type}}Config(blockTimestamp *big.Int) *{{.Contract.Type}}Config {
	return &{{.Contract.Type}}Config{
		UpgradeableConfig: UpgradeableConfig{
			BlockTimestamp: blockTimestamp,
			Disable:        true,
		},
	}
}

// Equal returns true if [s] is a [*{{.Contract.Type}}Config] and it has been configured identical to [c].
func (c *{{.Contract.Type}}Config) Equal(s StatefulPrecompileConfig) bool {
	// typecast before comparison
	other, ok := (s).(*{{.Contract.Type}}Config)
	if !ok {
		return false
	}
	// CUSTOM CODE STARTS HERE
	// modify this boolean accordingly with your custom {{.Contract.Type}}Config, to check if [other] and the current [c] are equal
	// if {{.Contract.Type}}Config contains only UpgradeableConfig {{if .Contract.AllowList}} and AllowListConfig {{end}} you can skip modifying it.
	equals := c.UpgradeableConfig.Equal(&other.UpgradeableConfig) {{if .Contract.AllowList}} && c.AllowListConfig.Equal(&other.AllowListConfig) {{end}}
	return equals
}

// String returns a string representation of the {{.Contract.Type}}Config.
func (c *{{.Contract.Type}}Config) String() string {
	bytes, _ := json.Marshal(c)
	return string(bytes)
}

// Address returns the address of the {{.Contract.Type}}. Addresses reside under the precompile/params.go
// Select a non-conflicting address and set it in the params.go.
func (c *{{.Contract.Type}}Config) Address() common.Address {
	return {{.Contract.Type}}Address
}

// Configure configures [state] with the initial configuration.
func (c *{{.Contract.Type}}Config) Configure(_ ChainConfig, state StateDB, _ BlockContext) {
	{{if .Contract.AllowList}}c.AllowListConfig.Configure(state, {{.Contract.Type}}Address){{end}}
	// CUSTOM CODE STARTS HERE
}

// Contract returns the singleton stateful precompiled contract to be used for {{.Contract.Type}}.
func (c *{{.Contract.Type}}Config) Contract() StatefulPrecompiledContract {
	return {{.Contract.Type}}Precompile
}

// Verify tries to verify {{.Contract.Type}}Config and returns an error accordingly.
func (c *{{.Contract.Type}}Config) Verify() error {
	{{if .Contract.AllowList}}
	// Verify AllowList first
	if err := c.AllowListConfig.Verify(); err != nil {
		return err
	}
	{{end}}
	// CUSTOM CODE STARTS HERE
	// Add your own custom verify code for {{.Contract.Type}}Config here
	// and return an error accordingly
	return nil
}

{{if .Contract.AllowList}}
// Get{{.Contract.Type}}AllowListStatus returns the role of [address] for the {{.Contract.Type}} list.
func Get{{.Contract.Type}}AllowListStatus(stateDB StateDB, address common.Address) AllowListRole {
	return getAllowListStatus(stateDB, {{.Contract.Type}}Address, address)
}

// Set{{.Contract.Type}}AllowListStatus sets the permissions of [address] to [role] for the
// {{.Contract.Type}} list. Assumes [role] has already been verified as valid.
// This stores the [role] in the contract storage with address [{{.Contract.Type}}Address]
// and [address] hash. It means that any reusage of the [address] key for different value
// conflicts with the same slot [role] is stored.
// Precompile implementations must use a different key than [address] for their storage.
func Set{{.Contract.Type}}AllowListStatus(stateDB StateDB, address common.Address, role AllowListRole) {
	setAllowListRole(stateDB, {{.Contract.Type}}Address, address, role)
}
{{end}}

{{range .Contract.Funcs}}
{{if len .Normalized.Inputs | lt 1}}
// Unpack{{capitalise .Normalized.Name}}Input attempts to unpack [input] into the arguments for the {{capitalise .Normalized.Name}}Input{}
// assumes that [input] does not include selector (omits first 4 func signature bytes)
func Unpack{{capitalise .Normalized.Name}}Input(input []byte) ({{capitalise .Normalized.Name}}Input, error) {
	inputStruct := {{capitalise .Normalized.Name}}Input{}
	err := {{$contract.Type}}ABI.UnpackInputIntoInterface(&inputStruct, "{{.Original.Name}}", input)

	return inputStruct, err
}

// Pack{{.Normalized.Name}} packs [inputStruct] of type {{capitalise .Normalized.Name}}Input into the appropriate arguments for {{.Original.Name}}.
func Pack{{.Normalized.Name}}(inputStruct {{capitalise .Normalized.Name}}Input) ([]byte, error) {
	return {{$contract.Type}}ABI.Pack("{{.Original.Name}}", {{range .Normalized.Inputs}} inputStruct.{{capitalise .Name}}, {{end}})
}
{{else if len .Normalized.Inputs | eq 1 }}
{{$method := .}}
{{$input := index $method.Normalized.Inputs 0}}
// Unpack{{capitalise .Normalized.Name}}Input attempts to unpack [input] into the {{bindtype $input.Type $structs}} type argument
// assumes that [input] does not include selector (omits first 4 func signature bytes)
func Unpack{{capitalise .Normalized.Name}}Input(input []byte)({{bindtype $input.Type $structs}}, error) {
res, err := {{$contract.Type}}ABI.UnpackInput("{{$method.Original.Name}}", input)
if err != nil {
	return {{convertToNil $input.Type}}, err
}
unpacked := *abi.ConvertType(res[0], new({{bindtype $input.Type $structs}})).(*{{bindtype $input.Type $structs}})
return unpacked, nil
}

// Pack{{.Normalized.Name}} packs [{{decapitalise $input.Name}}] of type {{bindtype $input.Type $structs}} into the appropriate arguments for {{$method.Original.Name}}.
// the packed bytes include selector (first 4 func signature bytes).
// This function is mostly used for tests.
func Pack{{$method.Normalized.Name}}( {{decapitalise $input.Name}} {{bindtype $input.Type $structs}},) ([]byte, error) {
	return {{$contract.Type}}ABI.Pack("{{$method.Original.Name}}", {{decapitalise $input.Name}},)
}
{{else}}
// Pack{{.Normalized.Name}} packs the include selector (first 4 func signature bytes).
// This function is mostly used for tests.
func Pack{{.Normalized.Name}}() ([]byte, error) {
	return {{$contract.Type}}ABI.Pack("{{.Original.Name}}")
}
{{end}}

{{if len .Normalized.Outputs | lt 1}}
// Pack{{capitalise .Normalized.Name}}Output attempts to pack given [outputStruct] of type {{capitalise .Normalized.Name}}Output
// to conform the ABI outputs.
func Pack{{capitalise .Normalized.Name}}Output (outputStruct {{capitalise .Normalized.Name}}Output) ([]byte, error) {
	return {{$contract.Type}}ABI.PackOutput("{{.Original.Name}}",
		{{- range .Normalized.Outputs}}
		outputStruct.{{capitalise .Name}},
		{{- end}}
	)
}

{{else if len .Normalized.Outputs | eq 1 }}
{{$method := .}}
{{$output := index $method.Normalized.Outputs 0}}
// Pack{{capitalise .Normalized.Name}}Output attempts to pack given {{decapitalise $output.Name}} of type {{bindtype $output.Type $structs}}
// to conform the ABI outputs.
func Pack{{$method.Normalized.Name}}Output ({{decapitalise $output.Name}} {{bindtype $output.Type $structs}}) ([]byte, error) {
	return {{$contract.Type}}ABI.PackOutput("{{$method.Original.Name}}", {{decapitalise $output.Name}})
}
{{end}}

func {{decapitalise .Normalized.Name}}(accessibleState PrecompileAccessibleState, caller common.Address, addr common.Address, input []byte, suppliedGas uint64, readOnly bool) (ret []byte, remainingGas uint64, err error) {
	if remainingGas, err = deductGas(suppliedGas, {{.Normalized.Name}}GasCost); err != nil {
		return nil, 0, err
	}

	{{- if not .Original.IsConstant}}
	if readOnly {
		return nil, remainingGas, vmerrs.ErrWriteProtection
	}
 	{{- end}}

	{{- if len .Normalized.Inputs | eq 0}}
	// no input provided for this function
	{{else}}
	// attempts to unpack [input] into the arguments to the {{.Normalized.Name}}Input.
	// Assumes that [input] does not include selector
	// You can use unpacked [inputStruct] variable in your code
	inputStruct, err := Unpack{{capitalise .Normalized.Name}}Input(input)
	if err != nil{
		return nil, remainingGas, err
	}
	{{- end}}

	{{if not .Original.IsConstant | and $contract.AllowList}}
	// Allow list is enabled and {{.Normalized.Name}} is a state-changer function.
	// This part of the code restricts the function to be called only by enabled/admin addresses in the allow list.
	// You can modify/delete this code if you don't want this function to be restricted by the allow list.
	stateDB := accessibleState.GetStateDB()
	// Verify that the caller is in the allow list and therefore has the right to modify it
	callerStatus := getAllowListStatus(stateDB, {{$contract.Type}}Address, caller)
	if !callerStatus.IsEnabled() {
		return nil, remainingGas, fmt.Errorf("%w: %s", ErrCannot{{.Normalized.Name}}, caller)
	}
	// allow list code ends here.
  {{end}}
	// CUSTOM CODE STARTS HERE
	{{- if len .Normalized.Inputs | ne 0}}
	_ = inputStruct // CUSTOM CODE OPERATES ON INPUT
	{{- end}}

	{{- if len .Normalized.Outputs | eq 0}}
	// this function does not return an output, leave this one as is
	packedOutput := []byte{}
	{{- else}}
	{{- if len .Normalized.Outputs | lt 1}}
	var output {{capitalise .Normalized.Name}}Output // CUSTOM CODE FOR AN OUTPUT
	{{- else }}
	{{$output := index .Normalized.Outputs 0}}
	var output {{bindtype $output.Type $structs}} // CUSTOM CODE FOR AN OUTPUT
	{{- end}}
	packedOutput, err := Pack{{.Normalized.Name}}Output(output)
	if err != nil {
		return nil, remainingGas, err
	}
	{{- end}}

	// Return the packed output and the remaining gas
	return packedOutput, remainingGas, nil
}
{{end}}

{{- if .Contract.Fallback}}
{{- with .Contract.Fallback}}
// {{decapitalise $contract.Type}}Fallback executed if a function identifier does not match any of the available functions in a smart contract.
// This function cannot take an input or return an output.
func {{decapitalise $contract.Type}}Fallback (accessibleState PrecompileAccessibleState, caller common.Address, addr common.Address, _ []byte, suppliedGas uint64, readOnly bool) (ret []byte, remainingGas uint64, err error) {
	if remainingGas, err = deductGas(suppliedGas, {{$contract.Type}}FallbackGasCost); err != nil {
		return nil, 0, err
	}

	if readOnly {
		return nil, remainingGas, vmerrs.ErrWriteProtection
	}

	{{- if $contract.AllowList}}
	// Allow list is enabled and {{.Normalized.Name}} is a state-changer function.
	// This part of the code restricts the function to be called only by enabled/admin addresses in the allow list.
	// You can modify/delete this code if you don't want this function to be restricted by the allow list.
	stateDB := accessibleState.GetStateDB()
	// Verify that the caller is in the allow list and therefore has the right to modify it
	callerStatus := getAllowListStatus(stateDB, {{$contract.Type}}Address, caller)
	if !callerStatus.IsEnabled() {
		return nil, remainingGas, fmt.Errorf("%w: %s", Err{{$contract.Type}}CannotFallback, caller)
	}
	// allow list code ends here.
	{{- end}}

	// CUSTOM CODE STARTS HERE

	// Fallback can return data in output.
	// The returned data will not be ABI-encoded.
	// Instead it will be returned without modifications (not even padding).
	output := []byte{}
	// return raw output
	return output, remainingGas, nil
}
{{- end}}
{{- end}}

// create{{.Contract.Type}}Precompile returns a StatefulPrecompiledContract with getters and setters for the precompile.
{{if .Contract.AllowList}} // Access to the getters/setters is controlled by an allow list for [precompileAddr].{{end}}
func create{{.Contract.Type}}Precompile(precompileAddr common.Address) StatefulPrecompiledContract {
	var functions []*statefulPrecompileFunction
	{{- if .Contract.AllowList}}
	functions = append(functions, createAllowListFunctions(precompileAddr)...)
	{{- end}}

	{{range .Contract.Funcs}}
	method{{.Normalized.Name}}, ok := {{$contract.Type}}ABI.Methods["{{.Original.Name}}"]
	if !ok{
		panic("given method does not exist in the ABI")
	}
	functions = append(functions, newStatefulPrecompileFunction(method{{.Normalized.Name}}.ID, {{decapitalise .Normalized.Name}}))
	{{end}}

	{{- if .Contract.Fallback}}
	// Construct the contract with the fallback function.
	contract := newStatefulPrecompileWithFunctionSelectors({{decapitalise $contract.Type}}Fallback, functions)
	{{- else}}
	// Construct the contract with no fallback function.
	contract := newStatefulPrecompileWithFunctionSelectors(nil, functions)
	{{- end}}
	return contract
}
`
