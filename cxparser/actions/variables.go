package actions

import (
	"github.com/skycoin/cx/cx/ast"
	"github.com/skycoin/cx/cx/constants"
	"github.com/skycoin/cx/cx/globals"
	"github.com/skycoin/cx/cx/types"
)

// DeclareGlobalInPackage creates a global variable in a specified package
//
// If `doesInitialize` is true, then `initializer` is used to initialize the
// new variable.
//
// Input arguments description:
// 	prgrm - a CXProgram that contains all the data and arrays of the program.
// 	pkg - the package where the global will belong.
// 	declarator - contains the name of the global var.
// 	declaration_specifiers - contains the type build of the global.
// 	initializer - if doesInitialize is true then this contains the initial
// 				  value of the global.
// 	doesInitialize - true if global is initialized or given a value.
func DeclareGlobalInPackage(prgrm *ast.CXProgram, pkg *ast.CXPackage,
	declarator *ast.CXArgument, declaration_specifiers *ast.CXArgument,
	initializer []ast.CXExpression, doesInitialize bool) {
	if pkg == nil {
		var err error
		pkg, err = prgrm.GetCurrentPackage()
		if err != nil {
			panic(err)
		}
	}
	declaration_specifiers.Package = ast.CXPackageIndex(pkg.Index)

	// Treat the name a bit different whether it's defined already or not.
	if glbl, err := pkg.GetGlobal(prgrm, declarator.Name); err == nil {
		// The name is already defined.
		glblIdx := glbl.Index

		if glbl.Offset < 0 || glbl.Size == 0 || glbl.TotalSize == 0 {
			// then it was only added a reference to the symbol
			var offExpr []ast.CXExpression
			if declaration_specifiers.IsSlice { // TODO:PTR move branch in WritePrimary
				offExpr = WritePrimary(prgrm, declaration_specifiers.Type,
					make([]byte, types.POINTER_SIZE), true)
			} else {
				offExpr = WritePrimary(prgrm, declaration_specifiers.Type,
					make([]byte, declaration_specifiers.TotalSize), false)
			}

			offExprAtomicOp, err := prgrm.GetCXAtomicOpFromExpressions(offExpr, 0)
			if err != nil {
				panic(err)
			}

			offExprAtomicOpOutput := prgrm.GetCXArgFromArray(offExprAtomicOp.Outputs[0])
			prgrm.CXArgs[glblIdx].Offset = offExprAtomicOpOutput.Offset
			prgrm.CXArgs[glblIdx].PassBy = offExprAtomicOpOutput.PassBy
		}

		// Checking if something is supposed to be initialized
		// and if `initializer` actually contains something.
		// If `initializer` is `nil`, this means that an expression
		// equivalent to nil was used, such as `[]i32{}`.
		if doesInitialize && initializer != nil {
			initializerExpressionIdx := initializer[len(initializer)-1].Index
			initializerExpressionOperator := prgrm.GetFunctionFromArray(prgrm.CXAtomicOps[initializerExpressionIdx].Operator)
			// then we just re-assign offsets
			if initializerExpressionOperator == nil {
				// then it's a literal
				declaration_specifiers.Name = prgrm.CXArgs[glblIdx].Name
				declaration_specifiers.Offset = prgrm.CXArgs[glblIdx].Offset
				declaration_specifiers.PassBy = prgrm.CXArgs[glblIdx].PassBy
				declaration_specifiers.Package = prgrm.CXArgs[glblIdx].Package

				prgrm.CXArgs[glblIdx] = *declaration_specifiers
				prgrm.CXArgs[glblIdx].Index = glblIdx

				prgrm.CXAtomicOps[initializerExpressionIdx].AddInput(prgrm, prgrm.CXAtomicOps[initializerExpressionIdx].Outputs[0])
				prgrm.CXAtomicOps[initializerExpressionIdx].Outputs = nil
				prgrm.CXAtomicOps[initializerExpressionIdx].AddOutput(prgrm, ast.CXArgumentIndex(glblIdx))
				opIdx := prgrm.AddNativeFunctionInArray(ast.Natives[constants.OP_IDENTITY])
				prgrm.CXAtomicOps[initializerExpressionIdx].Operator = opIdx
				prgrm.CXAtomicOps[initializerExpressionIdx].Package = prgrm.CXArgs[glblIdx].Package

				//add intialization statements, to array
				prgrm.SysInitExprs = append(prgrm.SysInitExprs, initializer...)
			} else {
				// then it's an expression
				declaration_specifiers.Name = prgrm.CXArgs[glblIdx].Name
				declaration_specifiers.Offset = prgrm.CXArgs[glblIdx].Offset
				declaration_specifiers.PassBy = prgrm.CXArgs[glblIdx].PassBy
				declaration_specifiers.Package = prgrm.CXArgs[glblIdx].Package

				prgrm.CXArgs[glblIdx] = *declaration_specifiers
				prgrm.CXArgs[glblIdx].Index = glblIdx

				if initializer[len(initializer)-1].IsStructLiteral() {
					index := prgrm.AddCXAtomicOp(&ast.CXAtomicOperator{Outputs: []ast.CXArgumentIndex{ast.CXArgumentIndex(glblIdx)}, Operator: -1, Function: -1})
					initializer = StructLiteralAssignment(prgrm,
						[]ast.CXExpression{
							{
								Index: index,
								Type:  ast.CX_ATOMIC_OPERATOR,
							},
						},
						initializer,
					)
				} else {
					prgrm.CXAtomicOps[initializerExpressionIdx].Outputs = nil
					prgrm.CXAtomicOps[initializerExpressionIdx].AddOutput(prgrm, ast.CXArgumentIndex(glblIdx))
				}
				//add intialization statements, to array
				prgrm.SysInitExprs = append(prgrm.SysInitExprs, initializer...)
			}
		} else {
			// we keep the last value for now
			declaration_specifiers.Name = prgrm.CXArgs[glblIdx].Name
			declaration_specifiers.Offset = prgrm.CXArgs[glblIdx].Offset
			declaration_specifiers.PassBy = prgrm.CXArgs[glblIdx].PassBy
			declaration_specifiers.Package = prgrm.CXArgs[glblIdx].Package
			prgrm.CXArgs[glblIdx] = *declaration_specifiers
			prgrm.CXArgs[glblIdx].Index = glblIdx
		}
	} else {
		// then it hasn't been defined
		var offExpr []ast.CXExpression
		if declaration_specifiers.IsSlice { // TODO:PTR move branch in WritePrimary
			offExpr = WritePrimary(prgrm, declaration_specifiers.Type, make([]byte, types.POINTER_SIZE), true)
		} else {
			offExpr = WritePrimary(prgrm, declaration_specifiers.Type, make([]byte, declaration_specifiers.TotalSize), false)
		}

		// Checking if something is supposed to be initialized
		// and if `initializer` actually contains something.
		// If `initializer` is `nil`, this means that an expression
		// equivalent to nil was used, such as `[]i32{}`.
		if doesInitialize && initializer != nil {
			initializerExpressionIdx := initializer[len(initializer)-1].Index
			initializerExpressionOperator := prgrm.GetFunctionFromArray(prgrm.CXAtomicOps[initializerExpressionIdx].Operator)

			offExprAtomicOp, err := prgrm.GetCXAtomicOpFromExpressions(offExpr, 0)
			if err != nil {
				panic(err)
			}

			if initializerExpressionOperator == nil {
				// then it's a literal
				offExprAtomicOpOutput := prgrm.GetCXArgFromArray(offExprAtomicOp.Outputs[0])
				declaration_specifiers.Name = declarator.Name
				declaration_specifiers.ArgDetails.FileLine = declarator.ArgDetails.FileLine
				declaration_specifiers.Offset = offExprAtomicOpOutput.Offset
				declaration_specifiers.Size = offExprAtomicOpOutput.Size
				declaration_specifiers.TotalSize = offExprAtomicOpOutput.TotalSize
				declaration_specifiers.Package = ast.CXPackageIndex(pkg.Index)

				opIdx := prgrm.AddNativeFunctionInArray(ast.Natives[constants.OP_IDENTITY])
				prgrm.CXAtomicOps[initializerExpressionIdx].Operator = opIdx
				prgrm.CXAtomicOps[initializerExpressionIdx].AddInput(prgrm, prgrm.CXAtomicOps[initializerExpressionIdx].Outputs[0])
				prgrm.CXAtomicOps[initializerExpressionIdx].Outputs = nil
				declSpecIdx := prgrm.AddCXArgInArray(declaration_specifiers)
				prgrm.CXAtomicOps[initializerExpressionIdx].AddOutput(prgrm, declSpecIdx)

				pkg.AddGlobal(prgrm, declSpecIdx)
				//add intialization statements, to array
				prgrm.SysInitExprs = append(prgrm.SysInitExprs, initializer...)
			} else {
				// then it's an expression

				offExprAtomicOpOutput := prgrm.GetCXArgFromArray(offExprAtomicOp.Outputs[0])
				declaration_specifiers.Name = declarator.Name
				declaration_specifiers.ArgDetails.FileLine = declarator.ArgDetails.FileLine
				declaration_specifiers.Offset = offExprAtomicOpOutput.Offset
				declaration_specifiers.Size = offExprAtomicOpOutput.Size
				declaration_specifiers.TotalSize = offExprAtomicOpOutput.TotalSize
				declaration_specifiers.Package = ast.CXPackageIndex(pkg.Index)
				declSpecIdx := prgrm.AddCXArgInArray(declaration_specifiers)

				if initializer[len(initializer)-1].IsStructLiteral() {
					index := prgrm.AddCXAtomicOp(&ast.CXAtomicOperator{Outputs: []ast.CXArgumentIndex{declSpecIdx}, Operator: -1, Function: -1})
					initializer = StructLiteralAssignment(prgrm,
						[]ast.CXExpression{
							{
								Index: index,
								Type:  ast.CX_ATOMIC_OPERATOR,
							},
						},
						initializer,
					)
				} else {
					prgrm.CXAtomicOps[initializerExpressionIdx].Outputs = nil
					prgrm.CXAtomicOps[initializerExpressionIdx].AddOutput(prgrm, declSpecIdx)
				}

				pkg.AddGlobal(prgrm, declSpecIdx)
				//add intialization statements, to array
				prgrm.SysInitExprs = append(prgrm.SysInitExprs, initializer...)
			}
		} else {
			offExprAtomicOp, err := prgrm.GetCXAtomicOpFromExpressions(offExpr, 0)
			if err != nil {
				panic(err)
			}

			offExprAtomicOpOutput := prgrm.GetCXArgFromArray(offExprAtomicOp.Outputs[0])
			declaration_specifiers.Name = declarator.Name
			declaration_specifiers.ArgDetails.FileLine = declarator.ArgDetails.FileLine
			declaration_specifiers.Offset = offExprAtomicOpOutput.Offset
			declaration_specifiers.Size = offExprAtomicOpOutput.Size
			declaration_specifiers.TotalSize = offExprAtomicOpOutput.TotalSize
			declaration_specifiers.Package = ast.CXPackageIndex(pkg.Index)
			declSpecIdx := prgrm.AddCXArgInArray(declaration_specifiers)

			pkg.AddGlobal(prgrm, declSpecIdx)
		}
	}
}

// DeclareLocal() creates a local variable inside a function.
//
// Returns a list of expressions that contains the initialization, if any.
//
// Input arguments description:
// 	prgrm - a CXProgram that contains all the data and arrays of the program.
// 	declarator - contains the name of the var.
// 	declaration_specifiers - contains the type build of the var.
// 	initializer - if doesInitialize is true then this contains the initial
// 				  value of the var.
// 	doesInitialize - true if var is initialized or given a value.
func DeclareLocal(prgrm *ast.CXProgram, declarator *ast.CXArgument, declarationSpecifiers *ast.CXArgument,
	initializer []ast.CXExpression, doesInitialize bool) []ast.CXExpression {
	if globals.FoundCompileErrors {
		return nil
	}

	declarationSpecifiers.IsLocalDeclaration = true

	pkg, err := prgrm.GetCurrentPackage()
	if err != nil {
		panic(err)
	}

	declCXLine := ast.MakeCXLineExpression(prgrm, CurrentFile, LineNo, LineStr)
	// Declaration expression to handle the inline initialization.
	// For example, `var foo i32 = 11` needs to be divided into two expressions:
	// one that declares `foo`, and another that assigns 11 to `foo`
	decl := ast.MakeAtomicOperatorExpression(prgrm, nil)
	expressionIdx := decl.Index
	prgrm.CXAtomicOps[expressionIdx].Package = ast.CXPackageIndex(pkg.Index)

	declarationSpecifiers.Name = declarator.Name
	declarationSpecifiers.ArgDetails.FileLine = declarator.ArgDetails.FileLine
	declarationSpecifiers.Package = ast.CXPackageIndex(pkg.Index)
	declarationSpecifiers.PreviouslyDeclared = true
	declSpecIdx := prgrm.AddCXArgInArray(declarationSpecifiers)
	declarationSpecifiers = prgrm.GetCXArgFromArray(declSpecIdx)
	prgrm.CXAtomicOps[expressionIdx].AddOutput(prgrm, declSpecIdx)

	// Checking if something is supposed to be initialized
	// and if `initializer` actually contains something.
	// If `initializer` is `nil`, this means that an expression
	// equivalent to nil was used, such as `[]i32{}`.
	if doesInitialize && initializer != nil {
		initializerExpression, err := prgrm.GetCXAtomicOpFromExpressions(initializer, len(initializer)-1)
		if err != nil {
			panic(err)
		}
		initializerExpressionOperator := prgrm.GetFunctionFromArray(initializerExpression.Operator)
		// THEN it's a literal, e.g. var foo i32 = 10;
		// ELSE it's an expression with an operator
		if initializerExpressionOperator == nil {
			exprCXLine := ast.MakeCXLineExpression(prgrm, CurrentFile, LineNo, LineStr)

			// we need to create an expression that links the initializer expressions
			// with the declared variable
			expr := ast.MakeAtomicOperatorExpression(prgrm, ast.Natives[constants.OP_IDENTITY])
			cxExprAtomicOpIdx := expr.Index
			prgrm.CXAtomicOps[cxExprAtomicOpIdx].Package = ast.CXPackageIndex(pkg.Index)

			initOut := prgrm.GetCXArgFromArray(initializerExpression.Outputs[0])
			initOutIdx := initializerExpression.Outputs[0]
			// CX checks the output of an expression to determine if it's being passed
			// by value or by reference, so we copy this property from the initializer's
			// output, in case of something like var foo *i32 = &bar
			prgrm.CXArgs[declSpecIdx].PassBy = initOut.PassBy

			prgrm.CXAtomicOps[cxExprAtomicOpIdx].AddOutput(prgrm, declSpecIdx)
			prgrm.CXAtomicOps[cxExprAtomicOpIdx].AddInput(prgrm, initOutIdx)

			initializer[len(initializer)-1] = *exprCXLine
			initializer = append(initializer, *expr)

			return append([]ast.CXExpression{*declCXLine, *decl}, initializer...)
		} else {
			expr := initializer[len(initializer)-1]
			cxExprAtomicOp, err := prgrm.GetCXAtomicOp(expr.Index)
			if err != nil {
				panic(err)
			}

			// THEN the expression has outputs created from the result of
			// handling a dot notation initializer, and it needs to be replaced
			// ELSE we simply add it using `AddOutput`
			if len(cxExprAtomicOp.Outputs) > 0 {
				// declSpecIdx := prgrm.AddCXArgInArray(declarationSpecifiers)
				cxExprAtomicOp.Outputs = []ast.CXArgumentIndex{declSpecIdx}
			} else {
				cxExprAtomicOp.AddOutput(prgrm, declSpecIdx)
			}

			return append([]ast.CXExpression{*declCXLine, *decl}, initializer...)
		}
	} else {
		exprCXLine := ast.MakeCXLineExpression(prgrm, CurrentFile, LineNo, LineStr)
		// There is no initialization.
		expr := ast.MakeAtomicOperatorExpression(prgrm, nil)
		cxAtomicOpIdx := expr.Index
		prgrm.CXAtomicOps[cxAtomicOpIdx].Package = ast.CXPackageIndex(pkg.Index)

		prgrm.CXArgs[declSpecIdx].Name = declarator.Name
		prgrm.CXArgs[declSpecIdx].ArgDetails.FileLine = declarator.ArgDetails.FileLine
		prgrm.CXArgs[declSpecIdx].Package = ast.CXPackageIndex(pkg.Index)
		prgrm.CXArgs[declSpecIdx].PreviouslyDeclared = true
		prgrm.CXAtomicOps[cxAtomicOpIdx].AddOutput(prgrm, declSpecIdx)

		return []ast.CXExpression{*exprCXLine, *expr}
	}
}
