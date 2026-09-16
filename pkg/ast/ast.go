package ast

type Node interface {
	Pos() (int, int)
}

// LoopStmt represents while, for-range (exclusive upper bound), and for-each.
type LoopStmt struct {
	Kind, Name                        string
	Condition, Start, End, Collection Expr
	Body                              []Stmt
	Line, Col                         int
}

func (s *LoopStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *LoopStmt) stmtNode()       {}

type LoopControl struct {
	Kind      string
	Line, Col int
}

func (s *LoopControl) Pos() (int, int) { return s.Line, s.Col }
func (s *LoopControl) stmtNode()       {}

type MatchStmt struct {
	Value           Expr
	OKName, ErrName string
	OK, Err         []Stmt
	Line, Col       int
}

func (s *MatchStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *MatchStmt) stmtNode()       {}

type TopLevel interface {
	Node
	topLevelNode()
}

type Stmt interface {
	Node
	stmtNode()
}

type Expr interface {
	Node
	exprNode()
}

type Type interface {
	Node
	typeNode()
	String() string
}

// Module representa a unidade básica de compilação da v0.2
type Module struct {
	Name  string
	Decls []TopLevel
	Line  int
	Col   int
}

func (m *Module) Pos() (int, int) { return m.Line, m.Col }

// ImportDecl representa (import "caminho")
type ImportDecl struct {
	Path string
	Line int
	Col  int
}

func (i *ImportDecl) Pos() (int, int) { return i.Line, i.Col }
func (i *ImportDecl) topLevelNode()   {}

// StructDecl representa (struct Nome (fields (f1 T1) ...))
type StructDecl struct {
	Name   string
	Fields []Field
	Line   int
	Col    int
}

func (s *StructDecl) Pos() (int, int) { return s.Line, s.Col }
func (s *StructDecl) topLevelNode()   {}

type Field struct {
	Name string
	Type Type
	Line int
	Col  int
}

func (f *Field) Pos() (int, int) { return f.Line, f.Col }

// FuncDecl representa (fn nome (params ...) (returns T) (effects ...) [(on-err var ...)] (body ...))
type FuncDecl struct {
	Name       string
	Params     []Param
	ReturnType Type
	Effects    []string
	OnErrVar   string
	OnErrBody  []Stmt
	Body       []Stmt
	Line       int
	Col        int
}

func (f *FuncDecl) Pos() (int, int) { return f.Line, f.Col }
func (f *FuncDecl) topLevelNode()   {}

// RouteDecl representa (route METHOD PATH (params ...) (returns T) (effects ...) [(on-err var ...)] (body ...))
type RouteDecl struct {
	Method     string
	Path       string
	Params     []Param
	ReturnType Type
	Effects    []string
	OnErrVar   string
	OnErrBody  []Stmt
	Body       []Stmt
	Line       int
	Col        int
}

func (r *RouteDecl) Pos() (int, int) { return r.Line, r.Col }
func (r *RouteDecl) topLevelNode()   {}

type Param struct {
	Name string
	Type Type
	Line int
	Col  int
}

func (p *Param) Pos() (int, int) { return p.Line, p.Col }

// LetStmt representa (let nome Tipo expr)
type LetStmt struct {
	Name  string
	Type  Type
	Value Expr
	Line  int
	Col   int
}

func (s *LetStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *LetStmt) stmtNode()       {}

// SetStmt representa (set nome expr)
type SetStmt struct {
	Name  string
	Value Expr
	Line  int
	Col   int
}

func (s *SetStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *SetStmt) stmtNode()       {}

// ReturnStmt representa (return [expr])
type ReturnStmt struct {
	Value Expr // pode ser nil em (return)
	Line  int
	Col   int
}

func (s *ReturnStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *ReturnStmt) stmtNode()       {}

// IfStmt representa (if cond (then ...) [(else ...)])
type IfStmt struct {
	Condition Expr
	Then      []Stmt
	Else      []Stmt // pode ser vazio/nil se não houver ramo else
	Line      int
	Col       int
}

func (s *IfStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *IfStmt) stmtNode()       {}

// SpawnStmt representa (spawn (call func args...))
type SpawnStmt struct {
	Call *CallExpr
	Line int
	Col  int
}

func (s *SpawnStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *SpawnStmt) stmtNode()       {}

// SendStmt representa (send canal expr)
type SendStmt struct {
	Channel Expr
	Value   Expr
	Line    int
	Col     int
}

func (s *SendStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *SendStmt) stmtNode()       {}

// ExprStmt representa (do expr)
type ExprStmt struct {
	Expr Expr
	Line int
	Col  int
}

func (s *ExprStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *ExprStmt) stmtNode()       {}

// Literais
type IntLiteral struct {
	Value int64
	Raw   string
	Line  int
	Col   int
}

func (l *IntLiteral) Pos() (int, int) { return l.Line, l.Col }
func (l *IntLiteral) exprNode()       {}

type FloatLiteral struct {
	Value float64
	Raw   string
	Line  int
	Col   int
}

func (l *FloatLiteral) Pos() (int, int) { return l.Line, l.Col }
func (l *FloatLiteral) exprNode()       {}

type BoolLiteral struct {
	Value bool
	Line  int
	Col   int
}

func (l *BoolLiteral) Pos() (int, int) { return l.Line, l.Col }
func (l *BoolLiteral) exprNode()       {}

type StringLiteral struct {
	Value string
	Line  int
	Col   int
}

func (l *StringLiteral) Pos() (int, int) { return l.Line, l.Col }
func (l *StringLiteral) exprNode()       {}

// IdentExpr
type IdentExpr struct {
	Name string
	Line int
	Col  int
}

func (e *IdentExpr) Pos() (int, int) { return e.Line, e.Col }
func (e *IdentExpr) exprNode()       {}

// CallExpr representa chamada de função: (call func args...) ou diretamente (func args...)
type CallExpr struct {
	Func    string
	Args    []Expr
	HasCall bool
	Line    int
	Col     int
}

func (e *CallExpr) Pos() (int, int) { return e.Line, e.Col }
func (e *CallExpr) exprNode()       {}

// TryExpr representa (try expr)
type TryExpr struct {
	Expr Expr
	Line int
	Col  int
}

func (e *TryExpr) Pos() (int, int) { return e.Line, e.Col }
func (e *TryExpr) exprNode()       {}

// BinaryOpExpr representa (op expr1 expr2)
type BinaryOpExpr struct {
	Op    string
	Left  Expr
	Right Expr
	Line  int
	Col   int
}

func (e *BinaryOpExpr) Pos() (int, int) { return e.Line, e.Col }
func (e *BinaryOpExpr) exprNode()       {}

// RecvExpr representa (recv canal)
type RecvExpr struct {
	Channel Expr
	Line    int
	Col     int
}

func (e *RecvExpr) Pos() (int, int) { return e.Line, e.Col }
func (e *RecvExpr) exprNode()       {}

// Tipos
type PrimitiveType struct {
	Name string
	Line int
	Col  int
}

func (t *PrimitiveType) Pos() (int, int) { return t.Line, t.Col }
func (t *PrimitiveType) typeNode()       {}
func (t *PrimitiveType) String() string  { return t.Name }

type NamedType struct {
	Name string
	Line int
	Col  int
}

func (t *NamedType) Pos() (int, int) { return t.Line, t.Col }
func (t *NamedType) typeNode()       {}
func (t *NamedType) String() string  { return t.Name }

type AppliedType struct {
	Constructor string
	Args        []Type
	Line        int
	Col         int
}

func (t *AppliedType) Pos() (int, int) { return t.Line, t.Col }
func (t *AppliedType) typeNode()       {}
func (t *AppliedType) String() string {
	res := "(" + t.Constructor
	for _, arg := range t.Args {
		res += " " + arg.String()
	}
	res += ")"
	return res
}
