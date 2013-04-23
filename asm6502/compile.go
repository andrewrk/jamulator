package asm6502

import (
	"github.com/axw/gollvm/llvm"
	"os"
)

func (p *Program) Compile(filename string) error {
	llvm.InitializeNativeTarget()
	builder := llvm.NewBuilder()
	defer builder.Dispose()

	mod := llvm.NewModule("asm_module")
	helloWorldText := llvm.ConstString("Hello, world!", true)
	strGlobal := llvm.AddGlobal(mod, helloWorldText.Type(), "str")
	strGlobal.SetLinkage(llvm.PrivateLinkage)
	strGlobal.SetInitializer(helloWorldText)
	strGlobal.SetGlobalConstant(true)
	bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)
	putsType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{bytePointerType}, false)
	putsFn := llvm.AddFunction(mod, "puts", putsType)
	putsFn.SetFunctionCallConv(llvm.CCallConv)
	putsFn.SetLinkage(llvm.ExternalLinkage)
	mainType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{}, false)
	mainFn := llvm.AddFunction(mod, "main", mainType)
	mainFn.SetFunctionCallConv(llvm.CCallConv)
	entry := llvm.AddBasicBlock(mainFn, "entry")
	builder.SetInsertPointAtEnd(entry)
	strPtr := builder.CreatePointerCast(strGlobal, bytePointerType, "")
	builder.CreateCall(putsFn, []llvm.Value{strPtr}, "")
	builder.CreateRet(llvm.ConstInt(llvm.Int32Type(), 0, false))


	err := llvm.VerifyModule(mod, llvm.ReturnStatusAction)
	if err != nil { return err }

	engine, err := llvm.NewJITCompiler(mod, 3)
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

	fd, err := os.Create(filename)
	if err != nil { return err}

	err = llvm.WriteBitcodeToFile(mod, fd)
	if err != nil { return err }

	err = fd.Close()
	if err != nil { return err }
	return nil
}
