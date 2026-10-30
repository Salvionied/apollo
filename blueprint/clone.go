package blueprint

import "math/big"

// Clone returns a deep copy of bp. The copy does not share schema nodes,
// parameter lists, or integer bounds with the receiver, so later calls such
// as ApplyValidatorParams can edit the copy without changing bp.
func (bp *Blueprint) Clone() *Blueprint {
	if bp == nil {
		return nil
	}
	c := &schemaCloner{memo: make(map[*Schema]*Schema)}
	out := &Blueprint{
		Preamble:   clonePreamble(bp.Preamble),
		Validators: make([]Validator, len(bp.Validators)),
	}
	if bp.Definitions != nil {
		out.Definitions = make(map[string]*Schema, len(bp.Definitions))
		for name, schema := range bp.Definitions {
			out.Definitions[name] = c.schemaPtr(schema)
		}
	}
	for i := range bp.Validators {
		out.Validators[i] = c.validator(bp.Validators[i])
	}
	return out
}

type schemaCloner struct {
	memo map[*Schema]*Schema
}

func clonePreamble(p Preamble) Preamble {
	p.Compiler = cloneCompiler(p.Compiler)
	return p
}

func cloneCompiler(c *Compiler) *Compiler {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}

func (c *schemaCloner) validator(v Validator) Validator {
	if v.Datum != nil {
		datum := c.argument(*v.Datum)
		v.Datum = &datum
	}
	v.Redeemer = c.argumentSpec(v.Redeemer)
	if len(v.Parameters) > 0 {
		params := make([]Parameter, len(v.Parameters))
		for i := range v.Parameters {
			params[i] = c.parameter(v.Parameters[i])
		}
		v.Parameters = params
	}
	return v
}

func (c *schemaCloner) parameter(p Parameter) Parameter {
	p.Purpose = clonePurpose(p.Purpose)
	p.Schema = c.schemaVal(p.Schema)
	return p
}

func (c *schemaCloner) argumentSpec(a ArgumentSpec) ArgumentSpec {
	out := ArgumentSpec{}
	if a.Single != nil {
		single := c.argument(*a.Single)
		out.Single = &single
	}
	if len(a.OneOf) > 0 {
		out.OneOf = make([]Argument, len(a.OneOf))
		for i := range a.OneOf {
			out.OneOf[i] = c.argument(a.OneOf[i])
		}
	}
	return out
}

func (c *schemaCloner) argument(a Argument) Argument {
	a.Purpose = clonePurpose(a.Purpose)
	a.Schema = c.schemaVal(a.Schema)
	return a
}

func clonePurpose(p *PurposeSpec) *PurposeSpec {
	if p == nil {
		return nil
	}
	return &PurposeSpec{Purposes: append([]string(nil), p.Purposes...)}
}

func (c *schemaCloner) schemaPtr(s *Schema) *Schema {
	if s == nil {
		return nil
	}
	if cloned, ok := c.memo[s]; ok {
		return cloned
	}
	out := &Schema{}
	c.memo[s] = out
	*out = c.schemaVal(*s)
	return out
}

func (c *schemaCloner) schemaVal(s Schema) Schema {
	s.Minimum = cloneBigInt(s.Minimum)
	s.Maximum = cloneBigInt(s.Maximum)
	if s.Enum != nil {
		s.Enum = append([]string(nil), s.Enum...)
	}
	s.AnyOf = c.schemaSlice(s.AnyOf)
	s.Fields = c.schemaSlice(s.Fields)
	s.Items = c.items(s.Items)
	s.Keys = c.schemaPtr(s.Keys)
	s.Values = c.schemaPtr(s.Values)
	s.resolved = c.schemaPtr(s.resolved)
	return s
}

func (c *schemaCloner) schemaSlice(in []*Schema) []*Schema {
	if in == nil {
		return nil
	}
	out := make([]*Schema, len(in))
	for i := range in {
		out[i] = c.schemaPtr(in[i])
	}
	return out
}

func (c *schemaCloner) items(it *ItemsSpec) *ItemsSpec {
	if it == nil {
		return nil
	}
	out := &ItemsSpec{Single: c.schemaPtr(it.Single)}
	out.Tuple = c.schemaSlice(it.Tuple)
	return out
}

func cloneBigInt(n *big.Int) *big.Int {
	if n == nil {
		return nil
	}
	return new(big.Int).Set(n)
}
