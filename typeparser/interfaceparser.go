package typeparser

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/smanierre/typer-site/model"
	"github.com/smanierre/typer-site/util"
)

type interfaceVisitor struct {
	interfaces  []model.InterfaceRecord
	prevToken   ast.Node
	packageName string
}

func (iv *interfaceVisitor) Visit(node ast.Node) ast.Visitor {
	switch node.(type) {
	case *ast.File:
		f := node.(*ast.File)
		iv.packageName = f.Name.String()
	case *ast.InterfaceType:
		record := model.InterfaceRecord{}
		record.Package = iv.packageName
		record.Methods = []model.MethodRecord{}
		// Default the record ID to -1 since the database will handle it on insert.
		record.ID = -1
		previousIdent, ok := iv.prevToken.(*ast.Ident)
		if !ok { // This is an interface that is a parameter to a method
			return iv
		}
		record.Name = previousIdent.String()
		record.Implementable = true
		iface := node.(*ast.InterfaceType)
		for _, method := range iface.Methods.List {
			// If the method is unexported, then don't add it to the interface and also make the interface as un-implementable
			if len(method.Names) != 0 && !util.StartsWithUppercase(method.Names[0].Name) {
				record.Implementable = false
				continue
			}
			methodRecord := model.MethodRecord{}
			methodRecord.Package = iv.packageName
			if len(method.Names) == 0 { //Embedded interface
				switch method.Type.(type) {
				case *ast.Ident: //Embedded interface is within the same package
					ident := method.Type.(*ast.Ident)
					if !util.StartsWithUppercase(ident.Name) {
						record.Implementable = false
						continue
					}
					methodRecord.Name = fmt.Sprintf("%s.%s", methodRecord.Package, ident.Name)
					methodRecord.Parameters = []string{}
					methodRecord.ReturnValues = []string{}
					methodRecord.ReceiverID = -1
					methodRecord.ID = -1
					record.Methods = append(record.Methods, methodRecord)
					continue
				case *ast.SelectorExpr:
					selectorExpr := method.Type.(*ast.SelectorExpr)
					methodRecord.Name = fmt.Sprintf("%s.%s", selectorExpr.X, selectorExpr.Sel)
					methodRecord.Parameters = []string{}
					methodRecord.ReturnValues = []string{}
					methodRecord.ReceiverID = -1
					methodRecord.ID = -1
					record.Methods = append(record.Methods, methodRecord)
					continue
				}
			} else {
				methodRecord.Name = method.Names[0].String()
			}
			funcDec := method.Type.(*ast.FuncType)

			// Check to see how many parameters there are, and then parse if there are any
			if funcDec.Params.NumFields() == 0 {
				methodRecord.Parameters = []string{}
			} else {
				methodRecord.Parameters = parseFieldList(funcDec.Params.List)
			}
			// Check to see how many return values there are, then parse any that exist
			if funcDec.Results.NumFields() == 0 {
				methodRecord.ReturnValues = []string{}
			} else {
				methodRecord.ReturnValues = parseFieldList(funcDec.Results.List)
			}

			// Set ID = -1 since the database will handle assigning an ID.
			// Set the Receiver ID to -1 since that will be handled later on.
			methodRecord.ID = -1
			methodRecord.ReceiverID = -1
			record.Methods = append(record.Methods, methodRecord)
		}
		iv.interfaces = append(iv.interfaces, record)
	}
	if node != nil {
		iv.prevToken = node
	}
	return iv
}

//TODO: Refactor this to work recursively and just do one thinf for each type, exiting on an ident or selectorExpr
func parseFieldList(fields []*ast.Field) []string {
	parameters := []string{}

	for _, f := range fields {
		repeat := len(f.Names)
		switch f.Type.(type) {
		case *ast.Ident: //Regular identifier e.g. string
			param := f.Type.(*ast.Ident)
			appendMultiple(param.String(), repeat, &parameters)
		case *ast.SelectorExpr: //Custom type e.g. io.Writer
			selectorExpr := f.Type.(*ast.SelectorExpr)
			appendMultiple(fmt.Sprintf("%s.%s", selectorExpr.X, selectorExpr.Sel), repeat, &parameters)
		case *ast.StarExpr: //Pointer e.g. *int
			starExpr := f.Type.(*ast.StarExpr)
			switch starExpr.X.(type) {
			case *ast.SelectorExpr: //pointer type is a custom type e.g. io.Writer
				selectorExpr := starExpr.X.(*ast.SelectorExpr)
				appendMultiple(fmt.Sprintf("*%s.%s", selectorExpr.X, selectorExpr.Sel), repeat, &parameters)
			case *ast.ArrayType: //pointer type is a slice or array e.g. *[]
				arrayType := starExpr.X.(*ast.ArrayType)
				switch arrayType.Elt.(type) {
				case *ast.StarExpr: //pointer is to a slice of pointers e.g. *[]*
					starExpr := arrayType.Elt.(*ast.StarExpr)
					switch starExpr.X.(type) {
					case *ast.Ident: // Pointer type is builtin e.g. *[]*int
						appendMultiple(fmt.Sprintf("*[]*%s", starExpr.X), repeat, &parameters)
					case *ast.SelectorExpr: // Pointer type is a custom type e.g. *[]*io.Writer
						selectorExpr := starExpr.X.(*ast.SelectorExpr)
						appendMultiple(fmt.Sprintf("*[]*%s.%s", selectorExpr.X, selectorExpr.Sel), repeat, &parameters)
					default: //Catch and log any uncaught cases to add in
						fmt.Printf("Uncaught type in parsing type of *[]* method parameter: %T\n", starExpr.X)
					}
				case *ast.SelectorExpr: // pointer is to a slice of custom types e.g. *[]io.Writer
					selectorExpr := arrayType.Elt.(*ast.SelectorExpr)
					appendMultiple(fmt.Sprintf("*[]%s.%s", selectorExpr.X, selectorExpr.Sel), repeat, &parameters)
				case *ast.Ident: // pointer is to a slice of builtin types e.g. *[]int
					appendMultiple(fmt.Sprintf("*[]%s", arrayType.Elt), repeat, &parameters)
				default: //Catch and log any uncaught cases to add in
					fmt.Printf("Uncaught type in parsing type of pointer slice(*[]) method parameter: %T\n", arrayType.Elt)
				}
			case *ast.Ident: //pointer type is a builtin type e.g. *int
				appendMultiple(fmt.Sprintf("*%s", starExpr.X), repeat, &parameters)
			default: //Catch and log any uncaught cases to add in
				fmt.Printf("Uncaught type in parsing type of pointer method parameter: %T\n", starExpr.X)
			}
		case *ast.ArrayType: //Array e.g. []int
			arrayType := f.Type.(*ast.ArrayType)
			switch arrayType.Elt.(type) {
			case *ast.StarExpr: //Array of pointers
				starExpr := arrayType.Elt.(*ast.StarExpr)
				switch starExpr.X.(type) {
				case *ast.Ident: //Array of builtin pointers e.g. *int
					appendMultiple(fmt.Sprintf("[]*%s", starExpr.X), repeat, &parameters)
				case *ast.SelectorExpr: //Array of custom type pointers e.g. *io.Writer
					selectorExpr := starExpr.X.(*ast.SelectorExpr)
					appendMultiple(fmt.Sprintf("[]*%s.%s", selectorExpr.X, selectorExpr.Sel), repeat, &parameters)
				default: //Catch and log any uncaught cases to add in
					fmt.Printf("Uncaught type in parsing type of slice of pointers([]*) parameter: %T\n", starExpr.X)
				}
			case *ast.SelectorExpr: //Array of custom types e.g. io.Writer
				selectorExpr := arrayType.Elt.(*ast.SelectorExpr)
				appendMultiple(fmt.Sprintf("[]%s.%s", selectorExpr.X, selectorExpr.Sel), repeat, &parameters)
			case *ast.Ident: //Array of builting types e.g. int
				appendMultiple(fmt.Sprintf("[]%s", arrayType.Elt), repeat, &parameters)
			case *ast.ArrayType: //Array of Arrays e.g. [][]
				arrayType := arrayType.Elt.(*ast.ArrayType)
				switch arrayType.Elt.(type) {
				case *ast.Ident: //Array of builtin types e.g. [][]int
					appendMultiple(fmt.Sprintf("[][]%s", arrayType.Elt), repeat, &parameters)
				case *ast.SelectorExpr: //Array of custom types e.g. [][]io.Writer
					selectorExpr := arrayType.Elt.(*ast.SelectorExpr)
					appendMultiple(fmt.Sprintf("[][]%s.%s", selectorExpr.X, selectorExpr.Sel), repeat, &parameters)
				case *ast.StarExpr: //Array of pointers e.g. [][]*
					starExpr := arrayType.Elt.(*ast.StarExpr)
					switch starExpr.X.(type) {
					case *ast.SelectorExpr: //Array of pointers to custom types e.g. [][]*io.Writer
						selectorExpr := starExpr.X.(*ast.SelectorExpr)
						appendMultiple(fmt.Sprintf("[][]*%s.%s", selectorExpr.X, selectorExpr.Sel), repeat, &parameters)
					case *ast.Ident: //Array of pointers to builtin types e.g. [][]*int
						appendMultiple(fmt.Sprintf("[][]*%s", starExpr.X), repeat, &parameters)
					default: //Catch and log any uncaught cases to add in
						fmt.Printf("Uncaught type in parsing type of 2d slice of pointers([][]*) parameter: %T\n", starExpr.X)
					}
				case *ast.ArrayType:
					arrayType := arrayType.Elt.(*ast.ArrayType)
					switch arrayType.Elt.(type) {
					case *ast.Ident:
						appendMultiple(fmt.Sprintf("[][][]%s", arrayType.Elt), repeat, &parameters)
					default:
						fmt.Printf("Uncaught type in parsing type of 3d slice ([][][]) parameter: %T\n", arrayType.Elt)
					}
				default: //Catch and log any uncaught cases to add in
					fmt.Printf("Uncaught type in parsing type of 2d slice ([][]) parameter: %T\n", arrayType.Elt)
				}
			default: //Catch and log any uncaught cases to add in
				fmt.Printf("Uncaught type in parsing slice type parameter: %T\n", arrayType.Elt)
			}
		case *ast.InterfaceType: //Interface parameter, right now only empty interfaces are supported e.g. interface{}
			interfaceType := f.Type.(*ast.InterfaceType)
			if len(interfaceType.Methods.List) == 0 { // only going to handle empty interfaces for now. If you aren't requiring an empty interface, don't be lazy and just define it...
				appendMultiple("interface{}", repeat, &parameters)
			} else {
				fmt.Println("Detected non empty interface parameter, ignoring.")
			}
		case *ast.Ellipsis: //Variadic parameter e.g. ...int
			ellipsis := f.Type.(*ast.Ellipsis)
			switch ellipsis.Elt.(type) {
			case *ast.Ident: //Variadic parameter is a builtin type e.g. ...int
				appendMultiple(fmt.Sprintf("...%s", ellipsis.Elt), repeat, &parameters)
			case *ast.SelectorExpr: //Variadic parameter is a custom type e.g. ...io.Writer
				selectorExpr := ellipsis.Elt.(*ast.SelectorExpr)
				appendMultiple(fmt.Sprintf("...%s.%s", selectorExpr.X, selectorExpr.Sel), repeat, &parameters)
			case *ast.InterfaceType: //Variatic parameter is of type interface e.g. ...interface{}
				interfaceType := ellipsis.Elt.(*ast.InterfaceType)
				if len(interfaceType.Methods.List) == 0 {
					appendMultiple("...interface{}", repeat, &parameters)
				} else {
					fmt.Println("Detected non empty interface parameter, ignoring.")
				}
			case *ast.MapType:
				mapType := ellipsis.Elt.(*ast.MapType)
				var key, value string
				//Determine the string value of the key type
				switch mapType.Key.(type) {
				case *ast.Ident: //Key type is a regular type e.g. int
					key = fmt.Sprint(mapType.Key)
				case *ast.SelectorExpr: //Keytype is a custom type e.g. io.Writer
					selectorExpr := mapType.Key.(*ast.SelectorExpr)
					key = fmt.Sprintf("%s.%s", selectorExpr.X, selectorExpr.Sel)
				default: //Catch and log any unhandled cases to add in later
					fmt.Printf("Uncaught type in parsing variadic method parameter map key: %T\n", mapType.Key)
				}
				//Determine the string value of the value type
				switch mapType.Value.(type) {
				case *ast.Ident: //Value type is a regular type e.g. int
					value = fmt.Sprint(mapType.Value)
				case *ast.SelectorExpr: //Value type is a custom type e.g. io.Writer
					selectorExpr := mapType.Value.(*ast.SelectorExpr)
					value = fmt.Sprintf("%s.%s", selectorExpr.X, selectorExpr.Sel)
				case *ast.StarExpr: //Value type is a pointer e.g. *int
					starExpr := mapType.Value.(*ast.StarExpr)
					switch starExpr.X.(type) {
					case *ast.SelectorExpr: //Map value type is a pointer to a custom type e.g. *io.Writer
						selectorExpr := starExpr.X.(*ast.SelectorExpr)
						value = fmt.Sprintf("*%s.%s", selectorExpr.X, selectorExpr.Sel)
					case *ast.Ident: //Map value type is a pointer to a builtin type e.g. *int
						value = fmt.Sprintf("*%s", starExpr.X)
					default: //Catch and log any uncaught cases to add in
						fmt.Printf("Uncaught type in parsing variadic type of map parameter pointer value: %T\n", starExpr.X)
					}
				case *ast.InterfaceType:
					// Assuming interface is empty, if not then ignoring
					interfaceType := mapType.Value.(*ast.InterfaceType)
					if len(interfaceType.Methods.List) == 0 {
						value = "interface{}"
					}
				default: //Catch and llog any unhandled cases to add in later
					fmt.Printf("Uncaught type in parsing variadic method parameter map value: %T\n", mapType.Value)
				}
				appendMultiple(fmt.Sprintf("...map[%s]%s", key, value), repeat, &parameters)
			default:
				fmt.Printf("Uncaught type in parsing variadic parameter: %T\n", ellipsis.Elt)
			}
		case *ast.MapType: //Parameter is a map
			mapType := f.Type.(*ast.MapType)
			var key, value string
			//Determine the string value of the key type
			switch mapType.Key.(type) {
			case *ast.Ident: //Key type is a regular type e.g. int
				key = fmt.Sprint(mapType.Key)
			case *ast.SelectorExpr: //Keytype is a custom type e.g. io.Writer
				selectorExpr := mapType.Key.(*ast.SelectorExpr)
				key = fmt.Sprintf("%s.%s", selectorExpr.X, selectorExpr.Sel)
			default: //Catch and log any unhandled cases to add in later
				fmt.Printf("Uncaught type in parsing method parameter map key: %T\n", mapType.Key)
			}
			//Determine the string value of the value type
			switch mapType.Value.(type) {
			case *ast.Ident: //Value type is a regular type e.g. int
				value = fmt.Sprint(mapType.Value)
			case *ast.SelectorExpr: //Value type is a custom type e.g. io.Writer
				selectorExpr := mapType.Value.(*ast.SelectorExpr)
				value = fmt.Sprintf("%s.%s", selectorExpr.X, selectorExpr.Sel)
			case *ast.StarExpr: //Value type is a pointer e.g. *int
				starExpr := mapType.Value.(*ast.StarExpr)
				switch starExpr.X.(type) {
				case *ast.SelectorExpr: //Map value type is a pointer to a custom type e.g. *io.Writer
					selectorExpr := starExpr.X.(*ast.SelectorExpr)
					value = fmt.Sprintf("*%s.%s", selectorExpr.X, selectorExpr.Sel)
				case *ast.Ident: //Map value type is a pointer to a builtin type e.g. *int
					value = fmt.Sprintf("*%s", starExpr.X)
				default: //Catch and log any uncaught cases to add in
					fmt.Printf("Uncaught type in parsing type of map parameter pointer value: %T\n", starExpr.X)
				}
			default: //Catch and llog any unhandled cases to add in later
				fmt.Printf("Uncaught type in parsing method parameter map value: %T\n", mapType.Value)
			}
			appendMultiple(fmt.Sprintf("map[%s]%s", key, value), repeat, &parameters)
		case *ast.FuncType: //Paramater is a function
			funcType := f.Type.(*ast.FuncType)
			params := parseFieldList(funcType.Params.List)
			var returnValues []string
			if funcType.Results != nil {
				returnValues = parseFieldList(funcType.Results.List)
			}
			paramsString := strings.Join(params, ", ")
			returnValuesString := strings.Join(returnValues, ", ")
			if strings.Contains(returnValuesString, ",") {
				appendMultiple(fmt.Sprintf("func(%s) (%s)", paramsString, returnValuesString), repeat, &parameters)
			} else {
				appendMultiple(fmt.Sprintf("func(%s) %s", paramsString, returnValuesString), repeat, &parameters)
			}
		case *ast.ChanType: //Parameter is a channel
			chanType := f.Type.(*ast.ChanType)
			switch chanType.Value.(type) {
			case *ast.StructType: //Channel sends or receives a struct
				structType := chanType.Value.(*ast.StructType)
				if len(structType.Fields.List) == 0 {
					appendMultiple(fmt.Sprint("chan struct{}"), repeat, &parameters)
				}
			case *ast.Ident: //Channel of builtin types
				appendMultiple(fmt.Sprintf("%s", chanType.Value), repeat, &parameters)
			default: //Catch and log any uncaught cases to add in
				fmt.Printf("Uncaught type in parsing chan type: %T\n", chanType.Value)
			}
		default: //Catch and log any uncaught cases to add in
			fmt.Printf("Uncaught type in parsing method parameter: %T\n", f.Type)
		}
	}
	return parameters
}

func appendMultiple(s string, n int, params *[]string) {
	if n == 1 || n == 0 { //The 0 is to handle the return values in function parameters that are nameless
		*params = append(*params, s)
		return
	}
	for i := 0; i < n; i++ {
		*params = append(*params, s)
	}
}
