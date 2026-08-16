// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: BUSL-1.1

package paginator

import (
	"fmt"
	"testing"

	"github.com/hashicorp/nomad/ci"
	"github.com/hashicorp/nomad/nomad/mock"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/shoenig/test/must"
)

func TestTokenizer(t *testing.T) {
	ci.Parallel(t)

	j := mock.Job()

	cases := []struct {
		name      string
		tokenizer Tokenizer[*structs.Job]
		expected  string
	}{
		{
			name:      "ID",
			tokenizer: IDTokenizer[*structs.Job](""),
			expected:  fmt.Sprintf("%v", j.ID),
		},
		{
			name:      "Namespace.ID",
			tokenizer: NamespaceIDTokenizer[*structs.Job](""),
			expected:  fmt.Sprintf("%v.%v", j.Namespace, j.ID),
		},
		{
			name:      "CreateIndex.ID",
			tokenizer: CreateIndexAndIDTokenizer[*structs.Job](""),
			expected:  fmt.Sprintf("%v.%v", j.CreateIndex, j.ID),
		},
		{
			name:      "ModifyIndex",
			tokenizer: ModifyIndexTokenizer[*structs.Job](""),
			expected:  fmt.Sprintf("%d", j.ModifyIndex),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token, _ := tc.tokenizer(j)
			must.Eq(t, tc.expected, token)
		})
	}
}

func TestCreateIndexAndIDTokenizer(t *testing.T) {
	ci.Parallel(t)

	cases := []struct {
		name          string
		obj           *mockCreateIndexObject
		target        string
		expectedToken string
		expectedCmp   int
	}{
		{
			name:          "common index (less)",
			obj:           newMockCreateIndexObject(12, "aaa-bbb-ccc"),
			target:        "12.bbb-ccc-ddd",
			expectedToken: "12.aaa-bbb-ccc",
			expectedCmp:   -1,
		},
		{
			name:          "common index (greater)",
			obj:           newMockCreateIndexObject(12, "bbb-ccc-ddd"),
			target:        "12.aaa-bbb-ccc",
			expectedToken: "12.bbb-ccc-ddd",
			expectedCmp:   1,
		},
		{
			name:          "common index (equal)",
			obj:           newMockCreateIndexObject(12, "bbb-ccc-ddd"),
			target:        "12.bbb-ccc-ddd",
			expectedToken: "12.bbb-ccc-ddd",
			expectedCmp:   0,
		},
		{
			name:          "less index",
			obj:           newMockCreateIndexObject(12, "aaa-bbb-ccc"),
			target:        "89.aaa-bbb-ccc",
			expectedToken: "12.aaa-bbb-ccc",
			expectedCmp:   -1,
		},
		{
			name:          "greater index",
			obj:           newMockCreateIndexObject(89, "aaa-bbb-ccc"),
			target:        "12.aaa-bbb-ccc",
			expectedToken: "89.aaa-bbb-ccc",
			expectedCmp:   1,
		},
		{
			name:          "common index start (less)",
			obj:           newMockCreateIndexObject(12, "aaa-bbb-ccc"),
			target:        "102.aaa-bbb-ccc",
			expectedToken: "12.aaa-bbb-ccc",
			expectedCmp:   -1,
		},
		{
			name:          "common index start (greater)",
			obj:           newMockCreateIndexObject(102, "aaa-bbb-ccc"),
			target:        "12.aaa-bbb-ccc",
			expectedToken: "102.aaa-bbb-ccc",
			expectedCmp:   1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := CreateIndexAndIDTokenizer[*mockCreateIndexObject](tc.target)
			actualToken, actualCmp := fn(tc.obj)
			must.Eq(t, tc.expectedToken, actualToken)
			must.Eq(t, tc.expectedCmp, actualCmp)
		})
	}
}

func newMockCreateIndexObject(createIndex uint64, id string) *mockCreateIndexObject {
	return &mockCreateIndexObject{
		createIndex: createIndex,
		id:          id,
	}
}

type mockCreateIndexObject struct {
	createIndex uint64
	id          string
}

func (m *mockCreateIndexObject) GetCreateIndex() uint64 {
	return m.createIndex
}

func (m *mockCreateIndexObject) GetID() string {
	return m.id
}

func TestNamespaceIDTokenizer(t *testing.T) {
	ci.Parallel(t)

	cases := []struct {
		name          string
		obj           *mockNamespaceIDObject
		target        string
		expectedToken string
		expectedCmp   int
	}{
		{
			// Regression for hashicorp/nomad#28211: the separator "." must not
			// participate in ordering. Namespace "team" is a prefix of "team-a"
			// which continues with "-" (0x2D < "." 0x2E), so comparing the
			// joined token "team.j4" against "team-a.j1" as one string wrongly
			// returns +1. Comparing namespace first yields -1, matching the
			// memdb (Namespace, ID) compound index order.
			name:          "prefix namespace ordering (less)",
			obj:           newMockNamespaceIDObject("team", "j4"),
			target:        "team-a.j1",
			expectedToken: "team.j4",
			expectedCmp:   -1,
		},
		{
			// Symmetric counterpart: "team-a" sorts after "team".
			name:          "prefix namespace ordering (greater)",
			obj:           newMockNamespaceIDObject("team-a", "j1"),
			target:        "team.j9",
			expectedToken: "team-a.j1",
			expectedCmp:   1,
		},
		{
			name:          "common namespace id (less)",
			obj:           newMockNamespaceIDObject("team", "j1"),
			target:        "team.j2",
			expectedToken: "team.j1",
			expectedCmp:   -1,
		},
		{
			name:          "common namespace id (greater)",
			obj:           newMockNamespaceIDObject("team", "j1"),
			target:        "team.j0",
			expectedToken: "team.j1",
			expectedCmp:   1,
		},
		{
			name:          "common namespace id (equal)",
			obj:           newMockNamespaceIDObject("team", "j1"),
			target:        "team.j1",
			expectedToken: "team.j1",
			expectedCmp:   0,
		},
		{
			// An ID may itself contain ".", so only the first "." separates
			// namespace from ID.
			name:          "id containing separator",
			obj:           newMockNamespaceIDObject("team", "j1.a"),
			target:        "team.j1.a",
			expectedToken: "team.j1.a",
			expectedCmp:   0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn := NamespaceIDTokenizer[*mockNamespaceIDObject](tc.target)
			actualToken, actualCmp := fn(tc.obj)
			must.Eq(t, tc.expectedToken, actualToken)
			must.Eq(t, tc.expectedCmp, actualCmp)
		})
	}
}

func newMockNamespaceIDObject(namespace, id string) *mockNamespaceIDObject {
	return &mockNamespaceIDObject{
		namespace: namespace,
		id:        id,
	}
}

type mockNamespaceIDObject struct {
	namespace string
	id        string
}

func (m *mockNamespaceIDObject) GetNamespace() string {
	return m.namespace
}

func (m *mockNamespaceIDObject) GetID() string {
	return m.id
}
