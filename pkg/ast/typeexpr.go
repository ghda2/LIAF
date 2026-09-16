package ast

// TypeFromExpr converte uma expressao usada em posicao de tipo na Type
// correspondente. Construtores de tipo aparecem como chamadas no fonte —
// `(list Task)` chega aqui como CallExpr{Func: "list", Args: [Task]} — entao
// checker e codegen precisavam da mesma traducao. Vive aqui para os dois
// concordarem sobre o que e um argumento de tipo valido.
//
// Devolve ok=false para expressoes que nao descrevem um tipo.
func TypeFromExpr(e Expr) (Type, bool) {
	switch v := e.(type) {
	case *IdentExpr:
		switch v.Name {
		case "int", "float", "str", "bool", "void":
			return &PrimitiveType{Name: v.Name, Line: v.Line, Col: v.Col}, true
		}
		return &NamedType{Name: v.Name, Line: v.Line, Col: v.Col}, true

	case *CallExpr:
		arity, ok := typeConstructors[v.Func]
		if !ok || len(v.Args) != arity {
			return nil, false
		}
		args := make([]Type, 0, len(v.Args))
		for _, a := range v.Args {
			t, ok := TypeFromExpr(a)
			if !ok {
				return nil, false
			}
			args = append(args, t)
		}
		return &AppliedType{Constructor: v.Func, Args: args, Line: v.Line, Col: v.Col}, true
	}
	return nil, false
}

var typeConstructors = map[string]int{"list": 1, "chan": 1, "map": 2, "result": 2}
