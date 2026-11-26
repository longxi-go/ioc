// Copyright © 2026-present the longxi-go/ioc authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

// Status is the persisted order workflow state.
type Status string

const (
	Placed    Status = "placed"
	Preparing Status = "preparing"
	Ready     Status = "ready"
	Collected Status = "collected"
	Cancelled Status = "cancelled"
)

// Valid rejects unknown states before querying persistence.
func (s Status) Valid() bool {
	switch s {
	case Placed, Preparing, Ready, Collected, Cancelled:
		return true
	}
	return false
}

// CanMoveTo defines transitions independently of HTTP and persistence.
func (s Status) CanMoveTo(next Status) bool {
	switch s {
	case Placed:
		return next == Preparing || next == Cancelled
	case Preparing:
		return next == Ready || next == Cancelled
	case Ready:
		return next == Collected || next == Cancelled
	}
	return false
}
