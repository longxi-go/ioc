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

import (
	"crypto/sha256"
	"fmt"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/longxi-go/gosr/pkg/codegen/imports"
	gs "github.com/longxi-go/gosr/pkg/ioc"
)

const (
	embeddedFieldMarker    = "@"
	importAliasFormat      = "ioc%x"
	importAliasDigestBytes = 5
)

func typeAlias(path string) string {
	if path == contractsPackage {
		return contractsAlias
	}
	if path == contextPackage {
		return contextPackage
	}
	if path == beanPackage {
		return beanAlias
	}
	if path == catalogPackage {
		return catalogAlias
	}
	h := sha256.Sum256([]byte(path))
	return fmt.Sprintf(importAliasFormat, h[:importAliasDigestBytes])
}

func describeType(t types.Type) (typeSpec, error) {
	t = publicRepresentation(t)
	references := imports.New("", importAlias)
	code := references.Type(t)
	entries, err := references.List()
	if err != nil {
		return typeSpec{}, err
	}
	aliases := make(map[string]string, len(entries))
	for _, entry := range entries {
		aliases[entry.Path] = entry.Alias
	}
	return typeSpec{
		Code:    code,
		Imports: aliases,
		Hidden:  hiddenPackages(t),
	}, nil
}

func contract(t types.Type) gs.Contract {
	return gs.Contract(canonical(types.Unalias(t)))
}

func canonical(t types.Type) string {
	t = types.Unalias(t)
	switch v := t.(type) {
	case *types.Basic:
		return types.Typ[v.Kind()].Name()
	case *types.Pointer:
		return goPointer + canonical(v.Elem())
	case *types.Slice:
		return goSlice + canonical(v.Elem())
	case *types.Array:
		return goTypeArgumentsOpen + strconv.FormatInt(v.Len(), 10) + goTypeArgumentsClose + canonical(v.Elem())
	case *types.Map:
		return goMapOpen + canonical(v.Key()) + goTypeArgumentsClose + canonical(v.Elem())
	case *types.Chan:
		prefix := goChannel
		if v.Dir() == types.SendOnly {
			prefix = goSendChannel
		}
		if v.Dir() == types.RecvOnly {
			prefix = goReceiveChannel
		}
		return prefix + canonical(v.Elem())
	case *types.Named:
		name := v.Obj().Name()
		if v.Obj().Pkg() != nil {
			name = v.Obj().Pkg().Path() + goSelector + name
		}
		if v.TypeArgs() != nil && v.TypeArgs().Len() > 0 {
			var args []string
			for i := 0; i < v.TypeArgs().Len(); i++ {
				args = append(args, canonical(v.TypeArgs().At(i)))
			}
			name += goTypeArgumentsOpen + strings.Join(args, goTypeSeparator) + goTypeArgumentsClose
		}
		return name
	case *types.Struct:
		var fields []string
		for i := 0; i < v.NumFields(); i++ {
			f := v.Field(i)
			name := f.Name()
			if !f.Exported() && f.Pkg() != nil {
				name = f.Pkg().Path() + goSelector + name
			}
			embedded := ""
			if f.Embedded() {
				embedded = embeddedFieldMarker
			}
			fields = append(fields, embedded+name+goTypeSpace+canonical(f.Type())+goTypeSpace+strconv.Quote(v.Tag(i)))
		}
		return goStructOpen + strings.Join(fields, goMemberSeparator) + goBodyClose
	case *types.Interface:
		v.Complete()
		var methods []string
		for i := 0; i < v.NumMethods(); i++ {
			m := v.Method(i)
			name := m.Name()
			if !m.Exported() && m.Pkg() != nil {
				name = m.Pkg().Path() + goSelector + name
			}
			methods = append(methods, name+canonical(m.Type()))
		}

		sort.Strings(methods)
		return goInterfaceOpen + strings.Join(methods, goMemberSeparator) + goBodyClose
	case *types.Union:
		var terms []string
		for i := 0; i < v.Len(); i++ {
			term := v.Term(i)
			prefix := ""
			if term.Tilde() {
				prefix = goUnderlyingType
			}
			terms = append(terms, prefix+canonical(term.Type()))
		}
		sort.Strings(terms)
		return strings.Join(terms, goUnionSeparator)
	case *types.Signature:
		var params, results []string
		for i := 0; i < v.Params().Len(); i++ {
			value := canonical(v.Params().At(i).Type())
			if v.Variadic() && i == v.Params().Len()-1 {
				value = goVariadic + strings.TrimPrefix(value, goSlice)
			}
			params = append(params, value)
		}
		for i := 0; i < v.Results().Len(); i++ {
			results = append(results, canonical(v.Results().At(i).Type()))
		}
		return goFunctionOpen + strings.Join(params, goTypeSeparator) + goResultsOpen + strings.Join(results, goTypeSeparator) + goCallClose
	default:
		return types.TypeString(t, func(p *types.Package) string {
			return p.Path()
		})
	}
}

func leaf(s *shape) types.Type {
	for s.Elem != nil {
		s = s.Elem
	}
	return s.Type
}

func isContext(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == contextPackage && n.Obj().Name() == contextType
}

func hiddenPackages(t types.Type) []string {
	seen := map[types.Type]bool{}
	hidden := map[string]bool{}
	var visit typeVisitFunc
	visit = func(t types.Type) {
		if seen[t] {
			return
		}
		seen[t] = true
		switch v := t.(type) {
		case *types.Alias:
			if v.Obj().Pkg() != nil && !v.Obj().Exported() {
				hidden[v.Obj().Pkg().Path()] = true
			}
			for i := 0; i < v.TypeArgs().Len(); i++ {
				visit(v.TypeArgs().At(i))
			}
		case *types.Named:
			if v.Obj().Pkg() != nil && !v.Obj().Exported() {
				hidden[v.Obj().Pkg().Path()] = true
			}
			for i := 0; i < v.TypeArgs().Len(); i++ {
				visit(v.TypeArgs().At(i))
			}
		case *types.Pointer:
			visit(v.Elem())
		case *types.Slice:
			visit(v.Elem())
		case *types.Array:
			visit(v.Elem())
		case *types.Map:
			visit(v.Key())
			visit(v.Elem())
		case *types.Chan:
			visit(v.Elem())
		case *types.Struct:
			for i := 0; i < v.NumFields(); i++ {
				f := v.Field(i)
				if !f.Exported() && f.Pkg() != nil {
					hidden[f.Pkg().Path()] = true
				}
				visit(f.Type())
			}
		case *types.Interface:
			v.Complete()
			for i := 0; i < v.NumMethods(); i++ {
				m := v.Method(i)
				if !m.Exported() && m.Pkg() != nil {
					hidden[m.Pkg().Path()] = true
				}
				visit(m.Type())
			}
			for i := 0; i < v.NumEmbeddeds(); i++ {
				visit(v.EmbeddedType(i))
			}
		case *types.Signature:
			for i := 0; i < v.Params().Len(); i++ {
				visit(v.Params().At(i).Type())
			}
			for i := 0; i < v.Results().Len(); i++ {
				visit(v.Results().At(i).Type())
			}
		}
	}
	visit(t)
	var result []string
	for path := range hidden {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

// publicRepresentation removes private aliases without exposing their names.
// Public aliases remain useful when they deliberately name a private type.
func publicRepresentation(t types.Type) types.Type {
	switch v := t.(type) {
	case *types.Alias:
		if !v.Obj().Exported() {
			return publicRepresentation(types.Unalias(v))
		}
		return t
	case *types.Pointer:
		return types.NewPointer(publicRepresentation(v.Elem()))
	case *types.Slice:
		return types.NewSlice(publicRepresentation(v.Elem()))
	case *types.Array:
		return types.NewArray(publicRepresentation(v.Elem()), v.Len())
	case *types.Map:
		return types.NewMap(publicRepresentation(v.Key()), publicRepresentation(v.Elem()))
	case *types.Chan:
		return types.NewChan(v.Dir(), publicRepresentation(v.Elem()))
	case *types.Signature:
		if v.TypeParams().Len() > 0 {
			return t
		}
		tuple := func(old *types.Tuple) *types.Tuple {
			vars := make([]*types.Var, old.Len())
			for i := range vars {
				p := old.At(i)
				vars[i] = types.NewVar(p.Pos(), p.Pkg(), p.Name(), publicRepresentation(p.Type()))
			}
			return types.NewTuple(vars...)
		}
		return types.NewSignatureType(nil, nil, nil, tuple(v.Params()), tuple(v.Results()), v.Variadic())
	case *types.Struct:
		fields := make([]*types.Var, v.NumFields())
		tags := make([]string, v.NumFields())
		for i := range fields {
			f := v.Field(i)
			fields[i] = types.NewField(f.Pos(), f.Pkg(), f.Name(), publicRepresentation(f.Type()), f.Embedded())
			tags[i] = v.Tag(i)
		}
		return types.NewStruct(fields, tags)
	case *types.Interface:
		v.Complete()
		methods := make([]*types.Func, v.NumMethods())
		for i := range methods {
			m := v.Method(i)
			sig := publicRepresentation(m.Type()).(*types.Signature)
			methods[i] = types.NewFunc(m.Pos(), m.Pkg(), m.Name(), sig)
		}
		return types.NewInterfaceType(methods, nil).Complete()
	case *types.Named:
		if v.TypeArgs().Len() > 0 {
			args := make([]types.Type, v.TypeArgs().Len())
			changed := false
			for i := range args {
				args[i] = publicRepresentation(v.TypeArgs().At(i))
				changed = changed || args[i] != v.TypeArgs().At(i)
			}
			if changed {
				if normalized, err := types.Instantiate(nil, v.Origin(), args, false); err == nil {
					return normalized
				}
			}
		}
		return t
	default:
		return t
	}
}
