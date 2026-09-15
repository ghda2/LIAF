package ast

type Node interface {
	Pos() (int, int)
}

type Stmt interface {
	Node
	stmtNode()
}

type Expr interface {
	Node
	exprNode()
}

type Program struct {
	Decls []TopLevelDecl
}

type TopLevelDecl interface {
	Node
	declNode()
}

// Param
type Param struct {
	Name string
	Type string
}

// FuncDecl represents [fn name (p1: t1) -> (ret_t) ... /fn name]
type FuncDecl struct {
	Name       string
	Params     []Param
	ReturnType string
	Body       []Stmt
	Line       int
	Col        int
}

func (f *FuncDecl) Pos() (int, int) { return f.Line, f.Col }
func (f *FuncDecl) declNode()       {}

// StructDecl represents [struct Name f1: t1 /struct Name]
type StructDecl struct {
	Name   string
	Fields []Param
	Line   int
	Col    int
}

func (s *StructDecl) Pos() (int, int) { return s.Line, s.Col }
func (s *StructDecl) declNode()       {}

// LetStmt represents [let x: int 10]
type LetStmt struct {
	Name  string
	Type  string
	Value Expr
	Line  int
	Col   int
}

func (s *LetStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *LetStmt) stmtNode()       {}

// ReturnStmt represents [return expr]
type ReturnStmt struct {
	Value Expr
	Line  int
	Col   int
}

func (s *ReturnStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *ReturnStmt) stmtNode()       {}

// SpawnStmt represents [spawn (fn_name args...)]
type SpawnStmt struct {
	Call *CallExpr
	Line int
	Col  int
}

func (s *SpawnStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *SpawnStmt) stmtNode()       {}

// SendStmt represents [send ch val]
type SendStmt struct {
	Channel Expr
	Value   Expr
	Line    int
	Col     int
}

func (s *SendStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *SendStmt) stmtNode()       {}

// IfStmt represents [if cond ... [else ... /else] /if]
type IfStmt struct {
	Condition Expr
	ThenBody  []Stmt
	ElseBody  []Stmt
	Line      int
	Col       int
}

func (s *IfStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *IfStmt) stmtNode()       {}

// ExprStmt wraps expression as a statement
type ExprStmt struct {
	Expression Expr
	Line       int
	Col        int
}

func (s *ExprStmt) Pos() (int, int) { return s.Line, s.Col }
func (s *ExprStmt) stmtNode()       {}

// CallExpr represents (func_name arg1 arg2)
type CallExpr struct {
	Fn   string
	Args []Expr
	Line int
	Col  int
}

func (e *CallExpr) Pos() (int, int) { return e.Line, e.Col }
func (e *CallExpr) exprNode()       {}

// RecvExpr represents [recv ch]
type RecvExpr struct {
	Channel Expr
	Line    int
	Col     int
}

func (e *RecvExpr) Pos() (int, int) { return e.Line, e.Col }
func (e *RecvExpr) exprNode()       {}
func (e *RecvExpr) stmtNode()       {}

// BinaryOpExpr represents (add a b), (sub a b), (eq a b), etc.
type BinaryOpExpr struct {
	Op    string
	Left  Expr
	Right Expr
	Line  int
	Col   int
}

func (e *BinaryOpExpr) Pos() (int, int) { return e.Line, e.Col }
func (e *BinaryOpExpr) exprNode()       {}

// Literals and Identifiers
type IdentifierExpr struct {
	Name string
	Line int
	Col  int
}

func (e *IdentifierExpr) Pos() (int, int) { return e.Line, e.Col }
func (e *IdentifierExpr) exprNode()       {}

type IntLiteral struct {
	Value int64
	Line  int
	Col   int
}

func (e *IntLiteral) Pos() (int, int) { return e.Line, e.Col }
func (e *IntLiteral) exprNode()       {}

type FloatLiteral struct {
	Value float64
	Line  int
	Col   int
}

func (e *FloatLiteral) Pos() (int, int) { return e.Line, e.Col }
func (e *FloatLiteral) exprNode()       {}

type StringLiteral struct {
	Value string
	Line  int
	Col   int
}

func (e *StringLiteral) Pos() (int, int) { return e.Line, e.Col }
func (e *StringLiteral) exprNode()       {}

type BoolLiteral struct {
	Value bool
	Line  int
	Col   int
}

func (e *BoolLiteral) Pos() (int, int) { return e.Line, e.Col }
func (e *BoolLiteral) exprNode()       {}
