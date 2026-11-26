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

package controller

import "github.com/longxi-go/ioc/examples/coffee/internal/model"

const (
	IndexPath           = "/"
	orderLocationFormat = "/orders/%d"
	apiLocationFormat   = "/api/orders/%d"
)

type workflowAction struct {
	Path   string
	Label  string
	Target model.Status
}

func workflow() []workflowAction {
	return []workflowAction{
		{
			Path:   "prepare",
			Label:  "Start preparation",
			Target: model.Preparing,
		},
		{
			Path:   "ready",
			Label:  "Mark ready",
			Target: model.Ready,
		},
		{
			Path:   "collect",
			Label:  "Collect coffee",
			Target: model.Collected,
		},
		{
			Path:   "cancel",
			Label:  "Cancel order",
			Target: model.Cancelled,
		},
	}
}
