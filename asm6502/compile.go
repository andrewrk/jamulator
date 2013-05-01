package asm6502

import (
	"github.com/axw/gollvm/llvm"
	"os"
	"fmt"
	"bytes"
)

type Compilation struct {
	program *Program
	mod llvm.Module
	builder llvm.Builder
	labeledData map[string] llvm.Value
	currentValue *bytes.Buffer
	currentLabel string
	Warnings []string
	Errors []string
}

func (c *Compilation) AddStmt(stmt *DataStatement) {
	if len(c.currentLabel) == 0 {
		// trash the data
		c.Warnings = append(c.Warnings, fmt.Sprintf("trashing data at 0x%04x", stmt.Offset))
		return
	}
	for _, item := range(stmt.dataList) {
		var err error
		switch v := item.(type) {
			case *IntegerDataItem:
				err = c.currentValue.WriteByte(byte(*v))
				if err != nil {
					c.Errors = append(c.Errors, err.Error())
					return
				}
			case *StringDataItem:
				_, err = c.currentValue.WriteString(string(*v))
				if err != nil {
					c.Errors = append(c.Errors, err.Error())
					return
				}
		}
	}
}

func (c *Compilation) Stop() {
	if len(c.currentLabel) == 0 { return }
	if c.currentValue.Len() == 0 { return }
	text := llvm.ConstString(c.currentValue.String(), false)
	strGlobal := llvm.AddGlobal(c.mod, text.Type(), c.currentLabel)
	strGlobal.SetLinkage(llvm.PrivateLinkage)
	strGlobal.SetInitializer(text)
	c.currentLabel = ""
}

func (c *Compilation) Start(stmt *LabeledStatement) {
	c.currentLabel = stmt.LabelName
	c.currentValue = new(bytes.Buffer)
}

func (c *Compilation) Visit(n Node) {
	switch t := n.(type) {
	case *DataStatement:
		c.AddStmt(t)
	case *LabeledStatement:
		c.Stop()
		c.Start(t)
	default:
		c.Stop()
	}
}

func (c *Compilation) VisitEnd(n Node) {}

func (p *Program) Compile(filename string) (c *Compilation) {
	llvm.InitializeNativeTarget()

	c = new(Compilation)
	c.program = p
	c.Warnings = []string{}
	c.Errors = []string{}
	c.mod = llvm.NewModule("asm_module")
	c.builder = llvm.NewBuilder()
	defer c.builder.Dispose()
	c.labeledData = map[string] llvm.Value{}
	p.Ast.Ast(c)


	// declare i32 @putchar(i32)
	putCharType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{llvm.Int32Type()}, false)
	putCharFn := llvm.AddFunction(c.mod, "putChar", putCharType)
	putCharFn.SetLinkage(llvm.ExternalLinkage)

	// declare void @exit(i32) noreturn nounwind
	exitType := llvm.FunctionType(llvm.VoidType(), []llvm.Type{llvm.Int32Type()}, false)
	exitFn := llvm.AddFunction(c.mod, "exit", exitType)
	exitFn.AddFunctionAttr(llvm.NoReturnAttribute|llvm.NoUnwindAttribute)
	exitFn.SetLinkage(llvm.ExternalLinkage)

	// main function / entry point
	mainType := llvm.FunctionType(llvm.Int32Type(), []llvm.Type{}, false)
	mainFn := llvm.AddFunction(c.mod, "main", mainType)
	mainFn.SetFunctionCallConv(llvm.CCallConv)
	entry := llvm.AddBasicBlock(mainFn, "entry")
	c.builder.SetInsertPointAtEnd(entry)
	c.builder.CreateUnreachable()

	//helloWorldText := llvm.ConstString("Hello, world!", true)
	//strGlobal := llvm.AddGlobal(mod, helloWorldText.Type(), "str")
	//strGlobal.SetLinkage(llvm.PrivateLinkage)
	//strGlobal.SetInitializer(helloWorldText)
	//bytePointerType := llvm.PointerType(llvm.Int8Type(), 0)

	err := llvm.VerifyModule(c.mod, llvm.ReturnStatusAction)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}

	engine, err := llvm.NewJITCompiler(c.mod, 3)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}
	defer engine.Dispose()

	pass := llvm.NewPassManager()
	defer pass.Dispose()

	pass.Add(engine.TargetData())
	pass.AddConstantPropagationPass()
	pass.AddInstructionCombiningPass()
	pass.AddPromoteMemoryToRegisterPass()
	pass.AddGVNPass()
	pass.AddCFGSimplificationPass()
	pass.Run(c.mod)

	c.mod.Dump()

	fd, err := os.Create(filename)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}

	err = llvm.WriteBitcodeToFile(c.mod, fd)
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}

	err = fd.Close()
	if err != nil {
		c.Errors = append(c.Errors, err.Error())
		return
	}

	return
}
