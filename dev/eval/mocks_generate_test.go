//go:build targ

package eval_test

//go:generate impgen eval.VaultCloner --dependency --import-path github.com/toejough/engram/dev/eval
//go:generate impgen eval.ConfigBuilder --dependency --import-path github.com/toejough/engram/dev/eval
//go:generate impgen eval.AgentRunner --dependency --import-path github.com/toejough/engram/dev/eval
//go:generate impgen eval.ResultsWriter --dependency --import-path github.com/toejough/engram/dev/eval
