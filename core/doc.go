// Package core is the cchef engine: the pieces an operation is written
// against and a recipe is run with.
//
// A [Dish] carries data between operations along with the type it is currently
// being treated as, so an operation that wants bytes and one that wants text
// can sit next to each other in the same recipe. An [Operation] declares what
// it is called and what arguments it takes ([OpMeta] and [ArgDef]) and
// transforms one Dish into the next. A [Recipe] is an ordered list of
// operations with their arguments; running it feeds each result into the next
// step.
//
// Operations are looked up by name in a [Registry]. The package-level
// [Default] registry is the one [Register] adds to and the one the cchef
// command uses; importing the ops package for its side effects fills it with
// every operation cchef implements.
//
//	import (
//		"github.com/roberson-io/cchef/core"
//		_ "github.com/roberson-io/cchef/ops" // register the operations
//	)
//
//	r, err := core.ParseRecipeConfig(`[{"op":"To Base64"}]`)
//	if err != nil {
//		return err
//	}
//	out, err := r.Execute(core.NewDish([]byte("hello"), core.TypeByteArray))
//	if err != nil {
//		return err
//	}
//	fmt.Println(out.String()) // aGVsbG8=
//
// Arguments may be given in full or left out. [DefaultArgs] fills in what an
// operation declares, and [CoerceArgs] converts and checks what a caller
// supplies against those declarations — the same validation the command-line
// interface applies, so a recipe behaves the same whichever way it is run.
//
// Registering an operation of your own is the same work cchef's own operations
// do: implement [Operation] and hand it to [Register]. It is then available by
// name to any recipe, alongside the built-in ones.
package core
