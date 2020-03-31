// Copyright 2019 Liquidata, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/liquidata-inc/dolt/go/libraries/doltcore/schema/typeinfo"
	"github.com/liquidata-inc/dolt/go/store/types"
)

var sch1 = mustSchema([]Column{
	strCol("a", 1, true),
	strCol("b", 2, false),
	strCol("c", 3, false),
})

var sch2 = mustSchema([]Column{
	strCol("aa", 1, true),
	strCol("dd", 4, false),
})

var sch3 = mustSchema([]Column{
	strCol("aaa", 1, true),
	strCol("bbb", 2, false),
	strCol("eee", 5, false),
})

var sch4 = mustSchema([]Column{
	strCol("a", 1, true),
	strCol("eeee", 5, false),
	strCol("ffff", 6, false),
})

var nameCollisionWithSch1 = mustSchema([]Column{
	strCol("a", 1, true),
	strCol("b", 22, false),
})

var tagCollisionWithSch1 = mustSchema([]Column{
	strCol("a", 1, true),
	{"collision", 2, types.IntKind, false, typeinfo.Int32Type, nil},
})

type SuperSchemaTest struct {
	// Name of the test
	Name string
	// Schemas to added to the SuperSchema
	Schemas []Schema
	// ExpectedSuperSchema to be created
	ExpectedSuperSchema SuperSchema
	// ExpectedGeneratedSchema generated by GenerateSchema()
	ExpectedGeneratedSchema Schema
	// Expected error message to be returned, if ay
	ExpectedErrString string
}

var SuperSchemaTests = []SuperSchemaTest{
	{
		Name:    "SuperSchema of one Schema",
		Schemas: []Schema{sch1},
		ExpectedSuperSchema: SuperSchema{
			allCols: mustColColl([]Column{
				strCol("", 1, true),
				strCol("", 2, false),
				strCol("", 3, false),
			}),
			tagNames: map[uint64][]string{1: {"a"}, 2: {"b"}, 3: {"c"}}},
		ExpectedGeneratedSchema: sch1,
	},
	{
		Name:    "SuperSchema of multiple Schemas",
		Schemas: []Schema{sch1, sch2, sch3},
		ExpectedSuperSchema: SuperSchema{
			allCols: mustColColl([]Column{
				strCol("", 1, true),
				strCol("", 2, false),
				strCol("", 3, false),
				strCol("", 4, false),
				strCol("", 5, false),
			}),
			tagNames: map[uint64][]string{1: {"aaa", "aa", "a"}, 2: {"bbb", "b"}, 3: {"c"}, 4: {"dd"}, 5: {"eee"}},
		},
		ExpectedGeneratedSchema: mustSchema([]Column{
			strCol("aaa", 1, true),
			strCol("bbb", 2, false),
			strCol("c", 3, false),
			strCol("dd", 4, false),
			strCol("eee", 5, false),
		}),
	},
	{
		Name:    "SuperSchema respects order of added Schemas",
		Schemas: []Schema{sch3, sch2, sch1},
		ExpectedSuperSchema: SuperSchema{
			allCols: mustColColl([]Column{
				strCol("", 1, true),
				strCol("", 2, false),
				strCol("", 5, false),
				strCol("", 4, false),
				strCol("", 3, false),
			}),
			tagNames: map[uint64][]string{1: {"a", "aa", "aaa"}, 2: {"b", "bbb"}, 3: {"c"}, 4: {"dd"}, 5: {"eee"}},
		},
		ExpectedGeneratedSchema: mustSchema([]Column{
			strCol("a", 1, true),
			strCol("b", 2, false),
			strCol("eee", 5, false),
			strCol("dd", 4, false),
			strCol("c", 3, false),
		}),
	},
	{
		Name:    "SuperSchema appends tag to disambiguate name collisions",
		Schemas: []Schema{sch1, nameCollisionWithSch1},
		ExpectedSuperSchema: SuperSchema{
			allCols: mustColColl([]Column{
				strCol("", 1, true),
				strCol("", 2, false),
				strCol("", 3, false),
				strCol("", 22, false),
			}),
			tagNames: map[uint64][]string{1: {"a"}, 2: {"b"}, 3: {"c"}, 22: {"b"}},
		},
		ExpectedGeneratedSchema: mustSchema([]Column{
			strCol("a", 1, true),
			strCol("b_2", 2, false),
			strCol("c", 3, false),
			strCol("b_22", 22, false),
		}),
	},
	{
		Name:              "SuperSchema errors on tag collision",
		Schemas:           []Schema{sch1, tagCollisionWithSch1},
		ExpectedErrString: "tag collision for columns b and collision, different definitions (tag: 2)",
	},
}

func TestSuperSchema(t *testing.T) {
	for _, test := range SuperSchemaTests {
		t.Run(test.Name, func(t *testing.T) {
			testSuperSchema(t, test)
		})
	}
	t.Run("SuperSchemaUnion", func(t *testing.T) {
		testSuperSchemaUnion(t)
	})
}

func testSuperSchema(t *testing.T, test SuperSchemaTest) {
	ss, err := NewSuperSchema(test.Schemas...)
	if test.ExpectedErrString != "" {
		assert.Error(t, err, test.ExpectedErrString)
	} else {
		require.NoError(t, err)

		assert.True(t, test.ExpectedSuperSchema.Equals(ss))
		// ensure Equals() method works by comparing SuperSchema internals
		superSchemaDeepEqual(t, &test.ExpectedSuperSchema, ss)

		// ensure naming works correctly in GenerateSchema()
		gs, err := ss.GenerateSchema()
		require.NoError(t, err)
		assert.Equal(t, test.ExpectedGeneratedSchema, gs)

		eq, err := SchemasAreEqual(test.ExpectedGeneratedSchema, gs)
		require.NoError(t, err)
		assert.True(t, eq)
	}
}

func testSuperSchemaUnion(t *testing.T) {
	ss12, err := NewSuperSchema(sch1, sch2)
	require.NoError(t, err)
	ss34, err := NewSuperSchema(sch3, sch4)
	require.NoError(t, err)

	unionSuperSchema, err := SuperSchemaUnion(ss12, ss34)
	require.NoError(t, err)
	expectedGeneratedSchema := mustSchema([]Column{
		strCol("a", 1, true),
		strCol("bbb", 2, false),
		strCol("c", 3, false),
		strCol("dd", 4, false),
		strCol("eeee", 5, false),
		strCol("ffff", 6, false),
	})
	gs, err := unionSuperSchema.GenerateSchema()
	require.NoError(t, err)
	assert.Equal(t, expectedGeneratedSchema, gs)

	// ensure that SuperSchemaUnion() respects order
	unionSuperSchema, err = SuperSchemaUnion(ss34, ss12)
	require.NoError(t, err)
	expectedGeneratedSchema = mustSchema([]Column{
		strCol("aa", 1, true),
		strCol("b", 2, false),
		strCol("eeee", 5, false),
		strCol("ffff", 6, false),
		strCol("c", 3, false),
		strCol("dd", 4, false),
	})
	gs, err = unionSuperSchema.GenerateSchema()
	require.NoError(t, err)
	assert.Equal(t, expectedGeneratedSchema, gs)
}

func superSchemaDeepEqual(t *testing.T, ss1, ss2 *SuperSchema) {
	assert.Equal(t, ss1.tagNames, ss2.tagNames)
	assert.Equal(t, *ss1.allCols, *ss2.allCols)
}

func mustSchema(cols []Column) Schema {
	return SchemaFromCols(mustColColl(cols))
}

func mustColColl(cols []Column) *ColCollection {
	cc, err := NewColCollection(cols...)
	if err != nil {
		panic(err)
	}
	return cc
}

func strCol(name string, tag uint64, isPK bool) Column {
	return Column{name, tag, types.StringKind, isPK, typeinfo.StringDefaultType, nil}
}
