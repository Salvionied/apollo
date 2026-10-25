package blueprint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// Blueprint is a parsed and validated CIP-57 Plutus contract blueprint.
type Blueprint struct {
	Preamble    Preamble           `json:"preamble"`
	Validators  []Validator        `json:"validators"`
	Definitions map[string]*Schema `json:"definitions,omitempty"`
}

// Preamble holds contract metadata from a blueprint.
type Preamble struct {
	Title         string    `json:"title"`
	Description   string    `json:"description,omitempty"`
	Version       string    `json:"version"`
	PlutusVersion string    `json:"plutusVersion"`
	Compiler      *Compiler `json:"compiler,omitempty"`
	License       string    `json:"license,omitempty"`
}

// Compiler identifies the compiler that produced a blueprint.
type Compiler struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// Validator describes one validator entry in a blueprint.
type Validator struct {
	Title        string       `json:"title"`
	Description  string       `json:"description,omitempty"`
	Datum        *Argument    `json:"datum,omitempty"`
	Redeemer     ArgumentSpec `json:"redeemer"`
	Parameters   []Parameter  `json:"parameters,omitempty"`
	CompiledCode string       `json:"compiledCode,omitempty"`
	Hash         string       `json:"hash,omitempty"`
}

// Parameter describes one compile-time validator parameter.
type Parameter struct {
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Purpose     PurposeSpec `json:"purpose,omitempty"`
	Schema      Schema      `json:"schema"`
}

// Argument describes a datum or single redeemer specification.
type Argument struct {
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Purpose     PurposeSpec `json:"purpose,omitempty"`
	Schema      Schema      `json:"schema"`
}

// ArgumentSpec holds either one Argument or a oneOf list of purpose-specific
// arguments.
type ArgumentSpec struct {
	Single *Argument
	OneOf  []Argument
}

// UnmarshalJSON decodes a single argument or a {"oneOf": [...]} argument set.
func (a *ArgumentSpec) UnmarshalJSON(raw []byte) error {
	var probe struct {
		OneOf []Argument `json:"oneOf"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return err
	}
	if len(probe.OneOf) > 0 {
		a.OneOf = probe.OneOf
		a.Single = nil
		return nil
	}
	var single Argument
	if err := json.Unmarshal(raw, &single); err != nil {
		return err
	}
	a.Single = &single
	a.OneOf = nil
	return nil
}

// PurposeSpec holds either one purpose string or a oneOf list of purposes.
type PurposeSpec struct {
	Purposes []string
}

// UnmarshalJSON decodes a string purpose or {"oneOf": [...]}.
func (p *PurposeSpec) UnmarshalJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		p.Purposes = []string{s}
		return nil
	}
	var wrapper struct {
		OneOf []string `json:"oneOf"`
	}
	if err := json.Unmarshal(trimmed, &wrapper); err != nil {
		return err
	}
	if len(wrapper.OneOf) == 0 {
		return errors.New("purpose oneOf must not be empty")
	}
	p.Purposes = wrapper.OneOf
	return nil
}

// ItemsSpec holds either a single list-element schema or a tuple of positional
// schemas.
type ItemsSpec struct {
	Single *Schema
	Tuple  []*Schema
}

// UnmarshalJSON decodes a schema object or an array of schema objects.
func (it *ItemsSpec) UnmarshalJSON(raw []byte) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return errors.New("items must not be null")
	}
	if trimmed[0] == '[' {
		var tuple []*Schema
		if err := json.Unmarshal(trimmed, &tuple); err != nil {
			return err
		}
		if len(tuple) == 0 {
			return errors.New("tuple items must not be empty")
		}
		it.Tuple = tuple
		return nil
	}
	var single Schema
	if err := json.Unmarshal(trimmed, &single); err != nil {
		return err
	}
	it.Single = &single
	return nil
}

// Schema is a CIP-57 Plutus Data Schema node.
type Schema struct {
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	DataType    string     `json:"dataType,omitempty"`
	Ref         string     `json:"$ref,omitempty"`
	AnyOf       []*Schema  `json:"anyOf,omitempty"`
	Index       *uint64    `json:"index,omitempty"`
	Fields      []*Schema  `json:"fields,omitempty"`
	Items       *ItemsSpec `json:"items,omitempty"`
	Keys        *Schema    `json:"keys,omitempty"`
	Values      *Schema    `json:"values,omitempty"`
	MinItems    *int       `json:"minItems,omitempty"`
	MaxItems    *int       `json:"maxItems,omitempty"`
	UniqueItems bool       `json:"uniqueItems,omitempty"`
	MinLength   *int       `json:"minLength,omitempty"`
	MaxLength   *int       `json:"maxLength,omitempty"`
	Enum        []string   `json:"enum,omitempty"`
	Minimum     *big.Int   `json:"minimum,omitempty"`
	Maximum     *big.Int   `json:"maximum,omitempty"`
}

// Parse decodes and validates a CIP-57 blueprint JSON document.
func Parse(raw []byte) (*Blueprint, error) {
	var bp Blueprint
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&bp); err != nil {
		return nil, fmt.Errorf("parse blueprint json: %w", err)
	}
	if err := bp.validate(); err != nil {
		return nil, err
	}
	return &bp, nil
}

// Validator returns the validator with the given title.
func (bp *Blueprint) Validator(title string) (*Validator, error) {
	for i := range bp.Validators {
		if bp.Validators[i].Title == title {
			return &bp.Validators[i], nil
		}
	}
	return nil, fmt.Errorf("validator %q not found", title)
}

// Definition returns the named schema from definitions.
func (bp *Blueprint) Definition(name string) (*Schema, error) {
	if bp.Definitions == nil {
		return nil, fmt.Errorf("definition %q not found", name)
	}
	s, ok := bp.Definitions[name]
	if !ok || s == nil {
		return nil, fmt.Errorf("definition %q not found", name)
	}
	return s, nil
}

// RedeemerSchema returns the redeemer schema for the given purpose. When the
// validator has a single redeemer without a purpose filter, purpose may be
// empty.
func (v *Validator) RedeemerSchema(purpose string) (*Schema, error) {
	if v.Redeemer.Single != nil {
		if purpose != "" && len(v.Redeemer.Single.Purpose.Purposes) > 0 && !containsPurpose(v.Redeemer.Single.Purpose.Purposes, purpose) {
			return nil, fmt.Errorf("validator %q redeemer does not support purpose %q", v.Title, purpose)
		}
		return &v.Redeemer.Single.Schema, nil
	}
	for i := range v.Redeemer.OneOf {
		arg := &v.Redeemer.OneOf[i]
		if containsPurpose(arg.Purpose.Purposes, purpose) {
			return &arg.Schema, nil
		}
	}
	return nil, fmt.Errorf("validator %q has no redeemer schema for purpose %q", v.Title, purpose)
}

func containsPurpose(purposes []string, target string) bool {
	for _, p := range purposes {
		if p == target {
			return true
		}
	}
	return false
}

func (bp *Blueprint) validate() error {
	if strings.TrimSpace(bp.Preamble.Title) == "" {
		return errors.New("preamble.title is required")
	}
	if strings.TrimSpace(bp.Preamble.PlutusVersion) != "" {
		switch strings.ToLower(bp.Preamble.PlutusVersion) {
		case "v1", "v2", "v3":
		default:
			return fmt.Errorf("unsupported preamble.plutusVersion %q", bp.Preamble.PlutusVersion)
		}
	}
	if len(bp.Validators) == 0 {
		return errors.New("blueprint must contain at least one validator")
	}

	for name, def := range bp.Definitions {
		if def == nil {
			return fmt.Errorf("definitions/%s: schema is null", name)
		}
		if err := bp.validateSchema(def, "definitions/"+name, make(map[string]bool)); err != nil {
			return err
		}
	}

	titles := make(map[string]struct{}, len(bp.Validators))
	for i := range bp.Validators {
		v := &bp.Validators[i]
		loc := fmt.Sprintf("validators[%d](%s)", i, v.Title)
		if strings.TrimSpace(v.Title) == "" {
			return fmt.Errorf("validators[%d]: title is required", i)
		}
		if _, dup := titles[v.Title]; dup {
			return fmt.Errorf("%s: duplicate validator title", loc)
		}
		titles[v.Title] = struct{}{}

		if v.Datum != nil {
			if err := validatePurposes(v.Datum.Purpose.Purposes, loc+"/datum"); err != nil {
				return err
			}
			if err := bp.validateSchema(&v.Datum.Schema, loc+"/datum", make(map[string]bool)); err != nil {
				return err
			}
		}
		if err := bp.validateRedeemer(&v.Redeemer, loc+"/redeemer"); err != nil {
			return err
		}
		for pIdx := range v.Parameters {
			pLoc := fmt.Sprintf("%s/parameters[%d]", loc, pIdx)
			if err := validatePurposes(v.Parameters[pIdx].Purpose.Purposes, pLoc); err != nil {
				return err
			}
			if err := bp.validateSchema(&v.Parameters[pIdx].Schema, pLoc, make(map[string]bool)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (bp *Blueprint) validateRedeemer(spec *ArgumentSpec, loc string) error {
	if spec.Single == nil && len(spec.OneOf) == 0 {
		return fmt.Errorf("%s: redeemer is required", loc)
	}
	if spec.Single != nil {
		if err := validatePurposes(spec.Single.Purpose.Purposes, loc); err != nil {
			return err
		}
		return bp.validateSchema(&spec.Single.Schema, loc, make(map[string]bool))
	}
	seenPurposes := make(map[string]struct{})
	for i := range spec.OneOf {
		arg := &spec.OneOf[i]
		itemLoc := fmt.Sprintf("%s/oneOf[%d]", loc, i)
		if len(arg.Purpose.Purposes) == 0 {
			return fmt.Errorf("%s: purpose is required inside redeemer oneOf", itemLoc)
		}
		if err := validatePurposes(arg.Purpose.Purposes, itemLoc); err != nil {
			return err
		}
		for _, p := range arg.Purpose.Purposes {
			if _, dup := seenPurposes[p]; dup {
				return fmt.Errorf("%s: overlapping purpose %q in redeemer oneOf", itemLoc, p)
			}
			seenPurposes[p] = struct{}{}
		}
		if err := bp.validateSchema(&arg.Schema, itemLoc, make(map[string]bool)); err != nil {
			return err
		}
	}
	return nil
}

func validatePurposes(purposes []string, loc string) error {
	seen := make(map[string]struct{}, len(purposes))
	for _, p := range purposes {
		switch p {
		case "spend", "mint", "withdraw", "publish":
		default:
			return fmt.Errorf("%s: invalid purpose %q", loc, p)
		}
		if _, dup := seen[p]; dup {
			return fmt.Errorf("%s: duplicate purpose %q", loc, p)
		}
		seen[p] = struct{}{}
	}
	return nil
}

func (bp *Blueprint) resolveRef(ref string) (string, *Schema, error) {
	const prefix = "#/definitions/"
	if !strings.HasPrefix(ref, prefix) {
		return "", nil, fmt.Errorf("invalid $ref %q: must start with %s", ref, prefix)
	}
	rawKey := strings.TrimPrefix(ref, prefix)
	key := strings.ReplaceAll(rawKey, "~1", "/")
	key = strings.ReplaceAll(key, "~0", "~")
	if bp.Definitions == nil {
		return "", nil, fmt.Errorf("unresolved $ref %q", ref)
	}
	target, ok := bp.Definitions[key]
	if !ok || target == nil {
		return "", nil, fmt.Errorf("unresolved $ref %q (definition %q)", ref, key)
	}
	return key, target, nil
}

func (bp *Blueprint) validateSchema(s *Schema, loc string, visiting map[string]bool) error {
	if s == nil {
		return fmt.Errorf("%s: schema is null", loc)
	}
	if s.Ref != "" {
		if s.DataType != "" || len(s.AnyOf) > 0 || s.Items != nil || s.Keys != nil || s.Values != nil || s.Fields != nil {
			return fmt.Errorf("%s: $ref cannot be combined with schema keywords", loc)
		}
		key, target, err := bp.resolveRef(s.Ref)
		if err != nil {
			return fmt.Errorf("%s: %w", loc, err)
		}
		if visiting[key] {
			return fmt.Errorf("%s: $ref cycle involving %q", loc, key)
		}
		visiting[key] = true
		err = bp.validateSchema(target, "definitions/"+key, visiting)
		delete(visiting, key)
		return err
	}

	if len(s.AnyOf) > 0 {
		if s.DataType != "" || s.Items != nil || s.Keys != nil || s.Values != nil || s.Fields != nil || s.Index != nil {
			return fmt.Errorf("%s: anyOf cannot be combined with dataType keywords", loc)
		}
		seenIndexes := make(map[uint64]struct{}, len(s.AnyOf))
		for i, variant := range s.AnyOf {
			vLoc := fmt.Sprintf("%s/anyOf[%d]", loc, i)
			if variant == nil || variant.DataType != "constructor" {
				return fmt.Errorf("%s: anyOf variants must have dataType constructor", vLoc)
			}
			if err := bp.validateSchema(variant, vLoc, visiting); err != nil {
				return err
			}
			if _, dup := seenIndexes[*variant.Index]; dup {
				return fmt.Errorf("%s: duplicate constructor index %d in anyOf", vLoc, *variant.Index)
			}
			seenIndexes[*variant.Index] = struct{}{}
		}
		return nil
	}

	if s.DataType == "" {
		// Opaque Plutus Data schema (for example definitions/Data).
		if err := rejectKeywordsFor(s, loc, ""); err != nil {
			return err
		}
		return nil
	}

	if strings.HasPrefix(s.DataType, "#") {
		return fmt.Errorf("%s: builtin dataType %q is not permitted on a Plutus data interface", loc, s.DataType)
	}

	switch s.DataType {
	case "integer":
		if err := rejectKeywordsFor(s, loc, "integer"); err != nil {
			return err
		}
		if s.Minimum != nil && s.Maximum != nil && s.Minimum.Cmp(s.Maximum) > 0 {
			return fmt.Errorf("%s: minimum exceeds maximum", loc)
		}
		return nil
	case "bytes":
		if err := rejectKeywordsFor(s, loc, "bytes"); err != nil {
			return err
		}
		if s.MinLength != nil && *s.MinLength < 0 {
			return fmt.Errorf("%s: minLength must be non-negative", loc)
		}
		if s.MaxLength != nil && *s.MaxLength < 0 {
			return fmt.Errorf("%s: maxLength must be non-negative", loc)
		}
		if s.MinLength != nil && s.MaxLength != nil && *s.MinLength > *s.MaxLength {
			return fmt.Errorf("%s: minLength exceeds maxLength", loc)
		}
		return nil
	case "list":
		if err := rejectKeywordsFor(s, loc, "list"); err != nil {
			return err
		}
		if s.Items == nil {
			return fmt.Errorf("%s: list schema requires items", loc)
		}
		if err := validateItemBounds(s.MinItems, s.MaxItems, loc); err != nil {
			return err
		}
		if s.Items.Single != nil {
			return bp.validateSchema(s.Items.Single, loc+"/items", visiting)
		}
		for i, elem := range s.Items.Tuple {
			if err := bp.validateSchema(elem, fmt.Sprintf("%s/items[%d]", loc, i), visiting); err != nil {
				return err
			}
		}
		return nil
	case "map":
		if err := rejectKeywordsFor(s, loc, "map"); err != nil {
			return err
		}
		if s.Keys == nil || s.Values == nil {
			return fmt.Errorf("%s: map schema requires keys and values", loc)
		}
		if err := validateItemBounds(s.MinItems, s.MaxItems, loc); err != nil {
			return err
		}
		if err := bp.validateSchema(s.Keys, loc+"/keys", visiting); err != nil {
			return err
		}
		return bp.validateSchema(s.Values, loc+"/values", visiting)
	case "constructor":
		if err := rejectKeywordsFor(s, loc, "constructor"); err != nil {
			return err
		}
		if s.Index == nil {
			return fmt.Errorf("%s: constructor schema requires index", loc)
		}
		if s.Fields == nil {
			return fmt.Errorf("%s: constructor schema requires fields", loc)
		}
		for i, f := range s.Fields {
			if err := bp.validateSchema(f, fmt.Sprintf("%s/fields[%d]", loc, i), visiting); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("%s: unknown dataType %q", loc, s.DataType)
	}
}

func validateItemBounds(minItems, maxItems *int, loc string) error {
	if minItems != nil && *minItems < 0 {
		return fmt.Errorf("%s: minItems must be non-negative", loc)
	}
	if maxItems != nil && *maxItems < 0 {
		return fmt.Errorf("%s: maxItems must be non-negative", loc)
	}
	if minItems != nil && maxItems != nil && *minItems > *maxItems {
		return fmt.Errorf("%s: minItems exceeds maxItems", loc)
	}
	return nil
}

func rejectKeywordsFor(s *Schema, loc, kind string) error {
	if kind != "list" && (s.Items != nil || s.UniqueItems) {
		return fmt.Errorf("%s: list keywords are not allowed on dataType %q", loc, kind)
	}
	if kind != "map" && (s.Keys != nil || s.Values != nil) {
		return fmt.Errorf("%s: map keywords are not allowed on dataType %q", loc, kind)
	}
	if kind != "list" && kind != "map" && (s.MinItems != nil || s.MaxItems != nil) {
		return fmt.Errorf("%s: item count keywords are not allowed on dataType %q", loc, kind)
	}
	if kind != "constructor" && (s.Index != nil || s.Fields != nil) {
		return fmt.Errorf("%s: constructor keywords are not allowed on dataType %q", loc, kind)
	}
	if kind != "bytes" && (s.MinLength != nil || s.MaxLength != nil || len(s.Enum) > 0) {
		return fmt.Errorf("%s: bytes keywords are not allowed on dataType %q", loc, kind)
	}
	if kind != "integer" && (s.Minimum != nil || s.Maximum != nil) {
		return fmt.Errorf("%s: integer keywords are not allowed on dataType %q", loc, kind)
	}
	return nil
}
