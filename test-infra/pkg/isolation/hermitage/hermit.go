package hermitage

import "github.com/pingcap/tipocket/test-infra/pkg/isolation/model"

var spec model.TestSpec

func init() {
	setup := `
		create table test (id int primary key, value int) engine=innodb;
		insert into test (id, value) values (1, 10), (2, 20);
	`
	teardown := `DROP TABLE test`

	stepS1a := `set session transaction isolation level repeatable read; begin;`
	stepS1b := `select * from test where value = 30;`    // should return nothing
	stepS1c := `select * from test where value % 3 = 0;` // -- T1. Still returns nothing
	stepS1d := `commit;`

	stepS2a := `set session transaction isolation level repeatable read; begin;`
	stepS2b := `insert into test (id, value) values(3, 30);`
	stepS2c := `commit`

	spec = model.TestSpec{
		SetupSQLs:   []string{setup},
		TeardownSQL: teardown,
		Sessions: []model.Session{
			{
				Name:        "s1",
				SetupSQL:    "",
				TeardownSQL: "",
				Steps: []model.Step{
					{
						Name:    "s1a",
						SQL:     stepS1a,
						Session: 0,
					},
					{
						Name:    "s1b",
						SQL:     stepS1b,
						Session: 0,
					},
					{
						Name:    "s1c",
						SQL:     stepS1c,
						Session: 0,
					},
					{
						Name:    "s1d",
						SQL:     stepS1d,
						Session: 0,
					},
				},
			},
			{
				Name:        "s2",
				SetupSQL:    "",
				TeardownSQL: "",
				Steps: []model.Step{
					{
						Name:    "s2a",
						SQL:     stepS2a,
						Session: 1,
					},
					{
						Name:    "s2b",
						SQL:     stepS2b,
						Session: 1,
					},
					{
						Name:    "s2c",
						SQL:     stepS2c,
						Session: 1,
					},
				},
			},
		},
		Permutations: []model.Permutation{
			{
				StepNames: []string{
					"s1a", "s2a", "s1b", "s2b", "s2c", "s1c", "s1d",
				},
			},
		},
		AllSteps: nil,
	}

	for _, v := range spec.Sessions {
		spec.AllSteps = append(spec.AllSteps, v.Steps...)
	}

	// TODO: sort all steps by name.
}
