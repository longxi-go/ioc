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

package compiler

func (e *emitter) listenerViews(recipe *recipe) []listenerView {
	if len(recipe.Listeners) == 0 {
		return nil
	}
	event := e.use(eventPackage, typeAlias(eventPackage)) + eventType
	views := make([]listenerView, 0, len(recipe.Listeners))
	for _, listener := range recipe.Listeners {
		view := listenerView{
			ID:             listener.ID,
			Event:          canonical(listener.Event),
			EventType:      e.typ(listener.Event),
			EventInterface: event,
			Receiver:       e.typ(recipe.Result),
			Method:         listener.Method,
			Source:         listener.Source,
			HasOrder:       listener.Order != nil,
		}
		if listener.Order != nil {
			view.Order = *listener.Order
		}
		views = append(views, view)
	}
	return views
}
