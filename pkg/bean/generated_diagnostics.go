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

package bean

// Generated adapter diagnostics are shared by all emitted packages. Classify
// failures through gosr error codes; these texts supply human-readable detail.
const (
	// PortCardinalityMismatchMessage identifies a violated generated single-value port.
	PortCardinalityMismatchMessage = "generated port cardinality mismatch"
	// PortTypeMismatchMessage identifies a value outside a generated port contract.
	PortTypeMismatchMessage = "generated port type mismatch"
	// ListenerReceiverMismatchMessage identifies a listener with an incompatible receiver.
	ListenerReceiverMismatchMessage = "listener receiver mismatch"
	// ListenerEventMismatchMessage identifies an event outside the selected listener contract.
	ListenerEventMismatchMessage = "listener event mismatch"
	// QueryNameCountMessage identifies an invalid generated query name list.
	QueryNameCountMessage = "at most one bean name is allowed"
	// UnsupportedQueryContractMessage identifies a query absent from the generated plan.
	UnsupportedQueryContractMessage = "contract is absent from generated application plan"
)
