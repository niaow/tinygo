//go:build !(gc.conservative || gc.conservative2 || gc.precise2 || gc.custom || gc.precise) || !tinygo.wasm

package task

type gcData struct{}

func (gcd *gcData) swap() {
}
