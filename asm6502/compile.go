package asm6502

import (
	"github.com/axw/gollvm/llvm"
)

func (p *Program) Compile(filename string) error {
	llvm.InitializeNativeTarget()
	builder := llvm.NewBuilder()
	defer builder.Dispose()

	mod := llvm.NewModule("asm_module")
	main_type := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{}, false)
	main_fn := llvm.AddFunction(mod, "main", main_type)
	main_fn.SetFunctionCallConv(llvm.CCallConv)
	entry := llvm.AddBasicBlock(main_fn, "entry")
	builder.SetInsertPointAtEnd(entry)
	builder.CreateRet(llvm.ConstInt(llvm.Int32Type(), 0, false))

	err := llvm.VerifyModule(mod, llvm.ReturnStatusAction)
	if err != nil { return err }

	engine, err := llvm.NewJITCompiler(mod, 2)
	if err != nil { return err }
	defer engine.Dispose()

	pass := llvm.NewPassManager()
	defer pass.Dispose()

	pass.Add(engine.TargetData())
	pass.AddConstantPropagationPass()
	pass.AddInstructionCombiningPass()
	pass.AddPromoteMemoryToRegisterPass()
	pass.AddGVNPass()
	pass.AddCFGSimplificationPass()
	pass.Run(mod)

	mod.Dump()
	return nil
}
