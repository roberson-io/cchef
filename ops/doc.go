// Package ops implements the CyberChef operations, each as a type satisfying
// [github.com/roberson-io/cchef/core.Operation].
//
// Every operation registers itself, so importing the package for its side
// effects is enough to make all of them available by name to a recipe:
//
//	import (
//		"github.com/roberson-io/cchef/core"
//		_ "github.com/roberson-io/cchef/ops"
//	)
//
//	r, _ := core.ParseRecipeConfig(`[{"op":"To Base64"}]`)
//	out, _ := r.Execute(core.NewDish([]byte("hello"), core.TypeByteArray))
//
// An operation can also be named directly, which the compiler can check where
// a lookup by name cannot. Its arguments are positional and in the order the
// operation declares them, so [github.com/roberson-io/cchef/core.DefaultArgs]
// is the easy way to fill in the ones you do not care about:
//
//	op := ops.ToBase64{}
//	out, err := op.Run(core.NewDish([]byte("hello"), core.TypeByteArray),
//		core.DefaultArgs(op.Args()))
//
// Run assumes its arguments have already been checked; pass them through
// [github.com/roberson-io/cchef/core.CoerceArgs] first when they come from
// somewhere untrusted, which is what recipe execution does.
//
// The operations follow CyberChef's behavior. Where cchef differs
// deliberately, PLAN.md records what and why.
package ops
