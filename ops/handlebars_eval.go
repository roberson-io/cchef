package ops

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/jsonval"
)

// hbContext is what a template is being rendered against: the value paths are
// looked up in, the value the whole render started from, the enclosing context
// for `../`, the `@` values a block sets, and the partials defined so far.
type hbContext struct {
	value    any
	root     any
	parent   *hbContext
	locals   map[string]any
	partials map[string][]hbNode
}

// child returns a context for the value inside a block, keeping the partials so
// that one defined outside stays usable within.
func (c *hbContext) child(value any, locals map[string]any) *hbContext {
	if locals == nil {
		locals = map[string]any{}
	}
	return &hbContext{
		value:    value,
		root:     c.root,
		parent:   c,
		locals:   locals,
		partials: c.partials,
	}
}

// lookup reads a path against the context. A path may name a field, walk into
// nested fields with dots, stand for the context itself, climb to the enclosing
// context with `../`, or read one of the `@` values a block sets.
func (c *hbContext) lookup(path string) any {
	path = strings.TrimSpace(path)

	if strings.HasPrefix(path, "@") {
		return c.lookupLocal(path)
	}

	ctx := c
	for strings.HasPrefix(path, "../") {
		if ctx.parent == nil {
			return nil
		}
		ctx = ctx.parent
		path = strings.TrimPrefix(path, "../")
	}

	if path == "." || path == "this" || path == "" {
		return ctx.value
	}
	path = strings.TrimPrefix(path, "this.")

	return hbWalk(ctx.value, path)
}

// lookupLocal reads one of the values a block makes available, which are named
// with a leading @. `@root` is the value the render started from and can be
// walked into like any other.
func (c *hbContext) lookupLocal(path string) any {
	// A value of an enclosing block is written @../name, with the climb inside
	// the @ rather than before it.
	remainder := strings.TrimPrefix(path, "@")
	ctx := c
	for strings.HasPrefix(remainder, "../") {
		if ctx.parent == nil {
			return nil
		}
		ctx = ctx.parent
		remainder = strings.TrimPrefix(remainder, "../")
	}

	name, rest, _ := strings.Cut(remainder, ".")

	if name == "root" {
		if rest == "" {
			return c.root
		}
		return hbWalk(c.root, rest)
	}

	// A block's own values are on the context it made; anything else is looked
	// for outwards, so a nested block can still read the one around it.
	for ; ctx != nil; ctx = ctx.parent {
		if v, ok := ctx.locals[name]; ok {
			if rest == "" {
				return v
			}
			return hbWalk(v, rest)
		}
	}
	return nil
}

// hbWalk follows a dotted path into a value.
func hbWalk(value any, path string) any {
	for part := range strings.SplitSeq(path, ".") {
		if part == "" || part == "this" {
			continue
		}
		switch container := value.(type) {
		case jsonval.Object:
			i := jsonval.Index(container, part)
			if i < 0 {
				return nil
			}
			value = container[i].V
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(container) {
				return nil
			}
			value = container[i]
		default:
			return nil
		}
	}
	return value
}

// hbMissingArgument names what each block helper says when it is written
// without the one value it works on.
//
//nolint:staticcheck,revive // Handlebars' verbatim error text
var hbMissingArgument = map[string]string{
	"if":     "#if requires exactly one argument",
	"unless": "#unless requires exactly one argument",
	"with":   "#with requires exactly one argument",
	"each":   "Must pass iterator to #each",
}

// render runs a block helper.
func (b *hbBlock) render(out *strings.Builder, ctx *hbContext) error {
	if b.arg == "" {
		if message, known := hbMissingArgument[b.helper]; known {
			return errors.New(message)
		}
	}
	switch b.helper {
	case "if":
		return b.renderConditional(out, ctx, hbTruthy(ctx.lookup(b.arg)))
	case "unless":
		return b.renderConditional(out, ctx, !hbTruthy(ctx.lookup(b.arg)))
	case "with":
		return b.renderWith(out, ctx)
	case "each":
		return b.renderEach(out, ctx)
	}
	//nolint:staticcheck,revive // Handlebars' verbatim error text
	return fmt.Errorf("Missing helper: %q", b.helper)
}

// renderConditional renders one side of the block.
func (b *hbBlock) renderConditional(out *strings.Builder, ctx *hbContext, yes bool) error {
	if yes {
		return hbRenderAll(out, b.body, ctx)
	}
	return hbRenderAll(out, b.inverse, ctx)
}

// renderWith renders the body against the named value, or the alternative when
// there is nothing there.
func (b *hbBlock) renderWith(out *strings.Builder, ctx *hbContext) error {
	value := ctx.lookup(b.arg)
	if !hbTruthy(value) {
		return hbRenderAll(out, b.inverse, ctx)
	}
	return hbRenderAll(out, b.body, ctx.child(value, nil))
}

// renderEach renders the body once per item of a list, or per field of an
// object, and the alternative when there is nothing to walk.
func (b *hbBlock) renderEach(out *strings.Builder, ctx *hbContext) error {
	switch collection := ctx.lookup(b.arg).(type) {
	case []any:
		if len(collection) == 0 {
			return hbRenderAll(out, b.inverse, ctx)
		}
		for i, item := range collection {
			locals := map[string]any{
				"index": float64(i),
				"first": i == 0,
				"last":  i == len(collection)-1,
			}
			if err := hbRenderAll(out, b.body, ctx.child(item, locals)); err != nil {
				return err
			}
		}
		return nil

	case jsonval.Object:
		// A block walks an object's fields in the order JavaScript enumerates
		// them: the ones named by a whole number first, in numeric order, then
		// the rest in the order they were written.
		fields := jsonval.ESOrder(collection)
		if len(fields) == 0 {
			return hbRenderAll(out, b.inverse, ctx)
		}
		for i, field := range fields {
			locals := map[string]any{
				"key":   field.K,
				"index": float64(i),
				"first": i == 0,
				"last":  i == len(fields)-1,
			}
			if err := hbRenderAll(out, b.body, ctx.child(field.V, locals)); err != nil {
				return err
			}
		}
		return nil
	}

	return hbRenderAll(out, b.inverse, ctx)
}
