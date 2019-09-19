// Copyright (c) 2018-2019 The Soteria DAG developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import "github.com/soteria-dag/soterd/chaincfg"

// MockAddr is an implementation of soterutil.Address that provides enough functionality to satisfy the
// createrawtransaction RPC call.
type MockAddr struct {
	value string
}

// String returns a string value of the MockAddr soterutil.Address interface implementation
func (ma *MockAddr) String() string {
	return ma.value
}

// EncodeAddress returns an empty value, because it's not meant to be used but is required to meet the soterutil.Address
// interface implementation.
func (ma *MockAddr) EncodeAddress() string {
	return ""
}

// ScriptAddress returns an empty byte value, because it's not meant to be used but is required to meet the
// soterutil.Address interface implementation.
func (ma *MockAddr) ScriptAddress() []byte {
	return []byte{}
}

// IsForNet always returns true, because it's not meant ot be used but is required to meet the soterutil.Address
// interface implementation.
func (ma *MockAddr) IsForNet(p *chaincfg.Params) bool {
	return true
}

func NewMockAddr(value string) *MockAddr {
	return &MockAddr{value: value}
}