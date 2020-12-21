package sqlbuilder

import (
	"fmt"
	"strings"
	"sync"
)

const (
	defaultPlaceholder         = "?"
	defaultExpressionDelimeter = ", "
)

type StatementKind uint

//go:generate stringer -type StatementKind -linecomment
const (
	_unknownStatement StatementKind = iota
	_SelectStatement                // select
)

type ClauseKind uint

// The order of these is how they must show up in SQL statements. The order is
// used as the slice index to guarantee the clauses show up in the right spot
// in the builder.
//go:generate stringer -type ClauseKind -linecomment
const (
	_unknownClause  ClauseKind = iota
	_FromClause                // from
	_JoinClause                // join
	_LeftJoinClause            // left join
	_WhereClause               // where
	_GroupByClause             // group by
	_OrderByClause             // order by
)

type Clause interface {
	Kind() ClauseKind
	Delimeter() string
	Expression
}

type fromClause struct {
	tables []Expression
}

func (c fromClause) Kind() ClauseKind  { return _FromClause }
func (c fromClause) Delimeter() string { return ", " }

func (c fromClause) Build() string {
	values := make([]string, len(c.tables))

	for i, e := range c.tables {
		values[i] = e.Build()
	}

	return strings.Join(values, defaultExpressionDelimeter)
}

type joinClause struct {
	table      Expression
	predicates []Expression
}

func (c joinClause) Kind() ClauseKind  { return _JoinClause }
func (c joinClause) Delimeter() string { return " " }

func (c joinClause) Build() string {
	values := make([]string, len(c.predicates))

	for i, e := range c.predicates {
		values[i] = e.Build()
	}

	return fmt.Sprintf("%s %s on %s",
		c.Kind().String(),
		c.table.Build(),
		strings.Join(values, " and "))
}

type leftJoinClause struct {
	table      Expression
	predicates []Expression
}

func (c leftJoinClause) Kind() ClauseKind  { return _LeftJoinClause }
func (c leftJoinClause) Delimeter() string { return " " }

func (c leftJoinClause) Build() string {
	values := make([]string, len(c.predicates))

	for i, e := range c.predicates {
		values[i] = e.Build()
	}

	return fmt.Sprintf("%s %s on %s",
		c.Kind().String(),
		c.table.Build(),
		strings.Join(values, " and "))
}

type whereClause struct {
	predicates MultiExpression
}

func (c whereClause) Kind() ClauseKind  { return _WhereClause }
func (c whereClause) Delimeter() string { return " and " }

func (c whereClause) Build() string {
	return Wrap(c.predicates).Build()
}

type groupByClause struct {
	columns []string
}

func (c groupByClause) Kind() ClauseKind  { return _GroupByClause }
func (c groupByClause) Delimeter() string { return ", " }

func (c groupByClause) Build() string {
	cols := strings.Join(c.columns, defaultExpressionDelimeter)
	return c.Kind().String() + " " + cols
}

type orderByClause struct {
	columns []string
}

func (c orderByClause) Kind() ClauseKind  { return _OrderByClause }
func (c orderByClause) Delimeter() string { return ", " }

func (c orderByClause) Build() string {
	cols := strings.Join(c.columns, defaultExpressionDelimeter)
	return c.Kind().String() + " " + cols
}

// TODO remove this. Must become an expression or statement. Currently exists
// to hack in window functions.
func OrderByC(cols ...string) Clause {
	return orderByClause{columns: cols}
}

type Expression interface {
	Build() string
}

type ExpressionFunc func() string

func (e ExpressionFunc) Build() string {
	return e()
}

func Ref(name string) ExpressionFunc {
	return func() string {
		return name
	}
}

func Const(value string) ExpressionFunc {
	return func() string {
		return "'" + value + "'"
	}
}

func As(expr Expression, alias string) ExpressionFunc {
	return func() string {
		return expr.Build() + " as " + Const(alias).Build()
	}
}

func RefAs(name, alias string) ExpressionFunc {
	return As(Ref(name), alias)
}

func Window(fn string, clause Clause) ExpressionFunc {
	return func() string {
		return fmt.Sprintf("%s over (%s)", fn, clause.Build())
	}
}

func Func(fn string, args ...Expression) ExpressionFunc {
	call := Ref(fn)
	me := MultiExpression{
		Delimeter:   defaultExpressionDelimeter,
		Expressions: args,
	}

	return func() string {
		return call.Build() + Wrap(me).Build()
	}
}

func Wrap(expr Expression) ExpressionFunc {
	return func() string {
		return "(" + expr.Build() + ")"
	}
}

func Columns(cols ...Expression) Expression {
	return MultiExpression{
		Delimeter:   defaultExpressionDelimeter,
		Expressions: cols,
	}
}

func Predicate(op string, left, right Expression) ExpressionFunc {
	return func() string {
		s := left.Build() + " " + op

		if right != nil {
			s += " " + right.Build()
		}

		return s
	}
}

func Equals(left, right Expression) ExpressionFunc {
	return Predicate("=", left, right)
}

func Greater(left, right Expression) ExpressionFunc {
	return Predicate(">", left, right)
}

func Less(left, right Expression) ExpressionFunc {
	return Predicate("<", left, right)
}

func GreaterOrEqual(left, right Expression) ExpressionFunc {
	return Predicate(">=", left, right)
}

func LessOrEqual(left, right Expression) ExpressionFunc {
	return Predicate("<=", left, right)
}

func In(left, right Expression) ExpressionFunc {
	return Predicate("in", left, Wrap(right))
}

func Like(left, right Expression) ExpressionFunc {
	return Predicate("like", left, right)
}

func NotLike(left, right Expression) ExpressionFunc {
	return Predicate("not like", left, right)
}

func Between(left, right Expression) ExpressionFunc {
	return Predicate("between", left, right)
}

func IsNull(expr Expression) ExpressionFunc {
	return Predicate("is null", expr, nil)
}

func IsNotNull(expr Expression) ExpressionFunc {
	return Predicate("is not null", expr, nil)
}

func Placeholder() ExpressionFunc {
	return func() string {
		return defaultPlaceholder
	}
}

type MultiExpression struct {
	Delimeter   string
	Expressions []Expression
}

func (e MultiExpression) Build() string {
	sl := SimpleListExpression{Delimeter: e.Delimeter}

	for _, expr := range e.Expressions {
		sl.Values = append(sl.Values, expr.Build())
	}

	return sl.Build()
}

type SimpleListExpression struct {
	Delimeter string
	Values    []string
}

func (e SimpleListExpression) Build() string {
	return strings.Join(e.Values, e.Delimeter)
}

type StatementOption func(*Statement)

type Statement struct {
	Kind        StatementKind
	Expressions []Expression
	Clauses     []Clause
}

func (s Statement) Build() string {
	builder := strings.Builder{}
	clauses := make([]*clauseBuilder, len(_ClauseKind_index))
	onceClauses := make(map[ClauseKind]*sync.Once)

	switch s.Kind {
	case _SelectStatement:
		onceClauses = map[ClauseKind]*sync.Once{
			_WhereClause: &sync.Once{},
			_FromClause:  &sync.Once{},
		}
	}

	builder.WriteString(s.Kind.String() + " ")

	for _, expr := range s.Expressions {
		builder.WriteString(expr.Build() + " ")
	}

	for _, clause := range s.Clauses {
		kind := clause.Kind()

		if clauses[kind] == nil {
			clauses[kind] = &clauseBuilder{
				kind: kind,
				me:   &MultiExpression{Delimeter: clause.Delimeter()},
			}
		}

		cb := clauses[kind]
		cb.me.Expressions = append(cb.me.Expressions, clause)
		clauses[kind] = cb
	}

	for _, group := range clauses {
		if group != nil {
			kind := group.kind

			if _, ok := onceClauses[kind]; ok {
				once := onceClauses[kind]
				once.Do(func() {
					builder.WriteString(kind.String() + " ")
				})
			}

			builder.WriteString(group.me.Build() + " ")
		}
	}

	return strings.TrimSpace(builder.String())
}

type clauseBuilder struct {
	kind ClauseKind
	me   *MultiExpression
}

// Select takes an expression as the column or columns and 0 or more options
// that modify the statement object to build the query.
func Select(columns Expression, opts ...StatementOption) Statement {
	st := Statement{
		Kind:        _SelectStatement,
		Expressions: []Expression{columns},
	}

	for _, opt := range opts {
		opt(&st)
	}

	return st
}

// From takes a list of expressions to use as a TableExpression list for the
// sql-from clause. The list is joined in argument order on ", ".
func From(tables ...Expression) StatementOption {
	return func(st *Statement) {
		st.Clauses = append(st.Clauses, fromClause{tables: tables})
	}
}

// FromSubselect takes a Statement and an optional as and returns it, wrapped in
// (), to the sql-from clause.
func FromSubselect(sub Statement, as string) StatementOption {
	expr := Wrap(sub)
	if as != "" {
		expr = As(expr, as)
	}

	return From(expr)
}

func Join(table Expression, predicates ...Expression) StatementOption {
	return func(st *Statement) {
		st.Clauses = append(st.Clauses, joinClause{
			table:      table,
			predicates: predicates,
		})
	}
}

func LeftJoin(table Expression, predicates ...Expression) StatementOption {
	return func(st *Statement) {
		st.Clauses = append(st.Clauses, leftJoinClause{
			table:      table,
			predicates: predicates,
		})
	}
}

// Where takes a list of expressions that are expected to be predicates of some
// kind. These are then join with " and " and wrapped in "()". Multiple uses of
// this StatementOption will only result in a single "where" clause with each
// distinct group of predicates wrapped in their own "()" and join with " and ".
func Where(predicates ...Expression) StatementOption {
	return func(st *Statement) {
		me := MultiExpression{
			Delimeter:   " and ",
			Expressions: predicates,
		}

		st.Clauses = append(st.Clauses, whereClause{predicates: me})
	}
}

// OrderBy takes a list of expressions and adds an order by clause to the
// statement.
//
// TODO make cols an expression (easier to add an expression func like RefDesc
// for sort direction.
func OrderBy(cols ...string) StatementOption {
	return func(st *Statement) {
		st.Clauses = append(st.Clauses, orderByClause{columns: cols})
	}
}

// GroupBy takes a list of expressions and adds an order by clause to the
// statement.
//
// TODO make cols an expression
func GroupBy(cols ...string) StatementOption {
	return func(st *Statement) {
		st.Clauses = append(st.Clauses, groupByClause{columns: cols})
	}
}
