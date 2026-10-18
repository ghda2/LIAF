package checker

import (
	"sort"

	"liaf/pkg/ast"
)

func (c *Checker) push() { c.scopes = append(c.scopes, map[string]*binding{}) }

func (c *Checker) pop() {
	scope := c.scopes[len(c.scopes)-1]
	ids := make([]string, 0, len(scope))
	for id := range scope {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		b := scope[id]
		if b.pending {
			c.error(b.node, "E_UNHANDLED_RESULT", id+" must be matched or returned")
		}
	}
	c.scopes = c.scopes[:len(c.scopes)-1]
}

func (c *Checker) define(id string, t ast.Type, n ast.Node, pending bool) {
	if id == "_" {
		return
	}
	s := c.scopes[len(c.scopes)-1]
	if s[id] != nil {
		c.error(n, "E_DUPLICATE_SYMBOL", id)
	}
	s[id] = &binding{t, n, pending}
}

func (c *Checker) lookup(id string) *binding {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		if b := c.scopes[i][id]; b != nil {
			return b
		}
	}
	return nil
}

func (c *Checker) consume(e ast.Expr) {
	if id, ok := e.(*ast.IdentExpr); ok {
		if b := c.lookup(id.Name); b != nil {
			b.pending = false
		}
	}
}

func (c *Checker) pending() map[*binding]bool {
	state := map[*binding]bool{}
	for _, s := range c.scopes {
		for _, b := range s {
			state[b] = b.pending
		}
	}
	return state
}

func restore(state map[*binding]bool) {
	for b, p := range state {
		b.pending = p
	}
}

func merge(a, b map[*binding]bool) {
	for binding, p := range a {
		binding.pending = p || b[binding]
	}
}
