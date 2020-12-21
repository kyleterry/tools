package sqlbuilder

import (
	"fmt"
	"testing"

	"github.com/matryer/is"
)

func TestSelect(t *testing.T) {
	cases := []struct {
		description string
		expected    string
		statement   Statement
	}{
		{
			description: "simple select",
			expected:    "select * from items where (id = ?) order by created_at",
			statement: Select(
				Columns(Ref("*")),
				From(Ref("items")),
				Where(Equals(Ref("id"), Placeholder())),
				OrderBy("created_at"),
			),
		},
		{
			description: "subselect table",
			expected:    "select * from (select id from items order by created_at)",
			statement: Select(
				Columns(Ref("*")),
				FromSubselect(Select(
					Columns(Ref("id")),
					From(Ref("items")),
					OrderBy("created_at"),
				), ""),
			),
		},
		{
			description: "simple select with is null and is not null",
			expected:    "select i.id from items as 'i' where (i.title is not null and i.content is null) order by i.created_at",
			statement: Select(
				Columns(Ref("i.id")),
				From(RefAs("items", "i")),
				Where(
					IsNotNull(Ref("i.title")),
					IsNull(Ref("i.content")),
				),
				OrderBy("i.created_at"),
			),
		},
		{
			description: "simple select with column funcions",
			expected:    "select id, coalesce(title, 'no title') as 'title' from items where (id = ?) order by created_at",
			statement: Select(
				Columns(
					Ref("id"),
					As(Func("coalesce", Ref("title"), Const("no title")), "title"),
				),
				From(Ref("items")),
				Where(Equals(Ref("id"), Placeholder())),
				OrderBy("created_at"),
			),
		},
	}

	is := is.New(t)
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			is.Equal(c.expected, c.statement.Build())
		})
	}
}

func BenchmarkStatementBuilder(b *testing.B) {
	st := Select(
		Columns(
			As(
				Window("row_number()", OrderByC("uu.id")), "row",
			),
			RefAs("uu.id", "id"),
			RefAs("uu.title", "id"),
			RefAs("u.id", "url.id"),
			RefAs("u.url", "url.url"),
			RefAs("u.title", "url.title"),
			RefAs("uu.user_id", "user.id"),
			RefAs("uu.favorite", "favorite"),
			RefAs("uu.created_at", "created_at"),
			RefAs("uu.updated_at", "updated_at"),
		),
		From(RefAs("user_urls", "uu")),
		Join(RefAs("urls", "u"), Equals(Ref("u.id"), Ref("uu.url_id"))),
		Where(
			Equals(Ref("uu.user_id"), Placeholder()),
		),
	)

	st2 := Select(
		Columns(Ref("uu.*")),
		FromSubselect(st, "uu"),
		LeftJoin(RefAs("user_url_tags", "ut"), Equals(Ref("ut.user_url_id"), Ref("uu.id"))),
		LeftJoin(RefAs("tags", "t"), Equals(Ref("t.id"), Ref("ut.tag_id"))),
		Where(
			Greater(Ref("uu.row"), Placeholder()),
			LessOrEqual(Ref("uu.row"), Placeholder()),
			In(Ref("t.name"), Placeholder()),
		),
		GroupBy("uu.id"),
	)

	for i := 0; i < b.N; i++ {
		st2.Build()
	}
}

func ExampleSelect() {
	st := Select(
		Columns(
			Ref("id"),
			RefAs("generated_name", "name"),
			As(Func("coalesce", Ref("location"), Const("earth")), "location"),
		),
		From(Ref("items")),
		Where(Equals(Ref("id"), Placeholder())),
		OrderBy("created_at"),
	)

	fmt.Println(st.Build())

	// Output: select id, generated_name as 'name', coalesce(location, 'earth') as 'location' from items where (id = ?) order by created_at
}

func ExampleFrom() {
	fmt.Println(Select(Ref("*"), From(RefAs("items", "i"))).Build())

	// Output: select * from items as 'i'
}

func ExampleFromSubselect() {
	sub := Select(Ref("1 + 1"))

	// subselect with no alias
	fmt.Println(Select(Ref("*"), FromSubselect(sub, "")).Build())
	// same subselect with alias
	fmt.Println(Select(Ref("*"), FromSubselect(sub, "result")).Build())

	// Output:
	// select * from (select 1 + 1)
	// select * from (select 1 + 1) as 'result'
}

func ExampleWhere() {
	where := Where(Equals(Ref("foo"), Placeholder()), Equals(Ref("bar"), Placeholder()))
	st := Select(Ref("*"), From(Ref("items")), where)

	fmt.Println(st.Build())

	// Output:
	// select * from items where (foo = ? and bar = ?)
}
